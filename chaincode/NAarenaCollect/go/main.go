package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// Nome da organização emissora (substitua pelo MSP ID real da sua organização emissora)
const EmissorMSP = "OrgEmissoraMSP"

// SmartContract define a estrutura do chaincode
type SmartContract struct {
	contractapi.Contract
}

// Estrutura do NFT
type NFT struct {
	ID             string `json:"id"`             // Identificador único do NFT
	Evento         string `json:"evento"`         // Nome do evento ou partida
	Estadio        string `json:"estadio"`        // Nome do estádio onde ocorreu
	ClubeCasa      string `json:"clubeCasa"`      // Nome do clube da casa
	ClubeVisitante string `json:"clubeVisitante"` // Nome do clube visitante
	Propriedade    string `json:"propriedade"`    // Proprietário atual do NFT
}

func (s *SmartContract) Init(ctx contractapi.TransactionContextInterface) error {
	fmt.Println("Chaincode SmartContract foi inicializado")
	return nil
}

// CriarNFT cria um novo NFT para um evento ou partida
func (s *SmartContract) CriarNFT(ctx contractapi.TransactionContextInterface, id, evento, estadio, clubeCasa, clubeVisitante, propriedade string) error {
	// Validação de parâmetros
	if id == "" || evento == "" || estadio == "" || clubeCasa == "" || clubeVisitante == "" || propriedade == "" {
		return fmt.Errorf("todos os campos devem ser preenchidos")
	}

	// Verificar se o cliente pertence à organização emissora
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("não foi possível recuperar o MSPID do cliente: %v", err)
	}

	if clientMSPID != EmissorMSP {
		return fmt.Errorf("apenas a organização emissora (%s) pode criar NFTs", EmissorMSP)
	}

	// Verificar se o NFT já existe
	nftAsBytes, err := ctx.GetStub().GetState(id)
	if err != nil {
		return fmt.Errorf("erro ao verificar a existência do NFT: %v", err)
	}
	if nftAsBytes != nil {
		return fmt.Errorf("NFT com ID %s já existe", id)
	}

	// Criar o NFT
	nft := NFT{
		ID:             id,
		Evento:         evento,
		Estadio:        estadio,
		ClubeCasa:      clubeCasa,
		ClubeVisitante: clubeVisitante,
		Propriedade:    propriedade,
	}

	nftAsBytes, err = json.Marshal(nft)
	if err != nil {
		return fmt.Errorf("erro ao serializar o NFT: %v", err)
	}

	err = ctx.GetStub().PutState(id, nftAsBytes)
	if err != nil {
		return fmt.Errorf("erro ao salvar o NFT no ledger: %v", err)
	}

	return nil
}

// ConsultarNFT retorna os detalhes de um NFT
func (s *SmartContract) ConsultarNFT(ctx contractapi.TransactionContextInterface, id string) (*NFT, error) {
	nftAsBytes, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("erro ao recuperar o NFT: %v", err)
	}
	if nftAsBytes == nil {
		return nil, fmt.Errorf("NFT com ID %s não encontrado", id)
	}

	var nft NFT
	err = json.Unmarshal(nftAsBytes, &nft)
	if err != nil {
		return nil, fmt.Errorf("erro ao deserializar o NFT: %v", err)
	}

	return &nft, nil
}

// TransferirNFT transfere a propriedade de um NFT entre usuários
func (s *SmartContract) TransferirNFT(ctx contractapi.TransactionContextInterface, id, novoProprietario string) error {
	if novoProprietario == "" {
		return fmt.Errorf("o novo proprietário não pode ser vazio")
	}

	nftAsBytes, err := ctx.GetStub().GetState(id)
	if err != nil {
		return fmt.Errorf("erro ao recuperar o NFT: %v", err)
	}
	if nftAsBytes == nil {
		return fmt.Errorf("NFT com ID %s não encontrado", id)
	}

	var nft NFT
	err = json.Unmarshal(nftAsBytes, &nft)
	if err != nil {
		return fmt.Errorf("erro ao deserializar o NFT: %v", err)
	}

	// Verificar se o cliente é o proprietário atual
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("não foi possível recuperar a identidade do cliente: %v", err)
	}

	if nft.Propriedade != clientID {
		return fmt.Errorf("somente o proprietário atual pode transferir este NFT")
	}

	// Atualizar o proprietário
	nft.Propriedade = novoProprietario
	nftAsBytes, err = json.Marshal(nft)
	if err != nil {
		return fmt.Errorf("erro ao serializar o NFT atualizado: %v", err)
	}

	err = ctx.GetStub().PutState(id, nftAsBytes)
	if err != nil {
		return fmt.Errorf("erro ao salvar o NFT atualizado no ledger: %v", err)
	}

	return nil
}

// ListarNFTs permite listar NFTs baseados no proprietário
func (s *SmartContract) ListarNFTs(ctx contractapi.TransactionContextInterface, proprietario string) ([]NFT, error) {
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, fmt.Errorf("erro ao listar NFTs: %v", err)
	}
	defer resultsIterator.Close()

	var nfts []NFT
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("erro ao iterar sobre os resultados: %v", err)
		}

		var nft NFT
		err = json.Unmarshal(queryResponse.Value, &nft)
		if err != nil {
			return nil, fmt.Errorf("erro ao deserializar NFT: %v", err)
		}

		if nft.Propriedade == proprietario {
			nfts = append(nfts, nft)
		}
	}

	return nfts, nil
}

func main() {
	smartContract := new(SmartContract)
	chaincode, err := contractapi.NewChaincode(smartContract)
	if err != nil {
		fmt.Printf("Erro ao criar o chaincode: %v\n", err)
		return
	}

	if err := chaincode.Start(); err != nil {
		fmt.Printf("Erro ao iniciar o chaincode: %v\n", err)
	}
}

func (s *SmartContract) Invoke(ctx contractapi.TransactionContextInterface) error {
	// Obtém o nome da função sendo chamada e os argumentos
	fn, args := ctx.GetStub().GetFunctionAndParameters()

	// Roteia para a função apropriada
	switch fn {
	case "CriarNFT":
		if len(args) < 6 {
			return fmt.Errorf("CriarNFT requer 6 argumentos: id, evento, estadio, clubeCasa, clubeVisitante, propriedade")
		}
		return s.CriarNFT(ctx, args[0], args[1], args[2], args[3], args[4], args[5])

	case "ConsultarNFT":
		if len(args) < 1 {
			return fmt.Errorf("ConsultarNFT requer 1 argumento: id")
		}
		nft, err := s.ConsultarNFT(ctx, args[0])
		if err != nil {
			return err
		}

		// Converte o NFT para JSON e retorna o resultado
		nftAsBytes, err := json.Marshal(nft)
		if err != nil {
			return fmt.Errorf("erro ao serializar o NFT: %v", err)
		}
		return ctx.GetStub().SetEvent("ConsultarNFTResult", nftAsBytes)

	case "TransferirNFT":
		if len(args) < 2 {
			return fmt.Errorf("TransferirNFT requer 2 argumentos: id, novoProprietario")
		}
		return s.TransferirNFT(ctx, args[0], args[1])

	case "ListarNFTs":
		if len(args) < 1 {
			return fmt.Errorf("ListarNFTs requer 1 argumento: proprietario")
		}
		nfts, err := s.ListarNFTs(ctx, args[0])
		if err != nil {
			return err
		}

		// Converte a lista de NFTs para JSON e retorna o resultado
		nftsAsBytes, err := json.Marshal(nfts)
		if err != nil {
			return fmt.Errorf("erro ao serializar os NFTs: %v", err)
		}
		return ctx.GetStub().SetEvent("ListarNFTsResult", nftsAsBytes)

	default:
		return fmt.Errorf("função desconhecida: %s", fn)
	}
}
