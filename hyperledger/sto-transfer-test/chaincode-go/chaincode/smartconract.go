package chaincode

import (
	"reflect"
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// SmartContract provides functions for managing Accounts and Transfers
type SmartContract struct {
	contractapi.Contract
}

// Insert struct field in alphabetic order => to achieve determinism across languages
// golang keeps the order when marshal to json but doesn't order automatically
type Account struct {
	Address        string     `json:"Address"`
	Fiat           float64    `json:"Fiat"`
	ST_1           float64    `json:"ST_1"`
}

type Transfer struct {
	FromAddress         string  `json:"FromAddress"`
	Price               float64 `json:"Price"`
	ST_ID               string  `json:"ST_ID"`
	Size                float64 `json:"Size"`
	TransferId          string  `json:"TransferId"`
	ToAddress           string  `json:"ToAddress"`
}

// InitLedger adds a base set of accounts to the ledger
func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	accounts := []Account{
		{Address: "admin", Fiat: 100000000, ST_1:10000},
		{Address: "user1", Fiat: 1000000, ST_1:20},
	}

	for _, account := range accounts {
		accountJSON, err := json.Marshal(account)
		if err != nil {
			return err
		}

		err = ctx.GetStub().PutState(account.Address, accountJSON)
		if err != nil {
			return fmt.Errorf("failed to put to world state. %v", err)
		}
	}

	return nil
}

func (s *SmartContract) ProcessTransferBatch(ctx contractapi.TransactionContextInterface, transferJSONBatchString string) (error) {
	accounts, accountErr := s.GetAllAccounts(ctx)
	if accountErr != nil {
		return fmt.Errorf("Error while getting accounts: %v", accountErr)
	}

    transfers, _ := s.unmarshalTransferBatchString(transferJSONBatchString)

	for _, transfer := range transfers {

		err := s.processTransfer(ctx, accounts, transfer)
		if err != nil {
			return fmt.Errorf("Error while processing transfer %s: %v", transfer.TransferId, err)
		}
	}

	for address, account := range accounts {
		err := s.UpdateAccountByObject(ctx, *account)
		if err != nil {
			return fmt.Errorf("Error while updating account %s: %v", address, err)
		}
	}
	return nil
}

func (s *SmartContract) processTransfer(ctx contractapi.TransactionContextInterface, accounts map[string]*Account, transfer Transfer) (error) {
	//verify balance
	err := s.verifySufficientBalance(ctx, accounts, transfer)
	if err != nil {
		return fmt.Errorf("Error while verifying transfer %s: %v", transfer.TransferId, err)
	}
	stID := transfer.ST_ID
	fromAccount, _ := accounts[transfer.FromAddress]
	toAccount, _ := accounts[transfer.ToAddress]
	// Use reflection to access the ST field based on stID
	fromSTField := reflect.ValueOf(fromAccount).Elem().FieldByName(stID)
	toSTField := reflect.ValueOf(toAccount).Elem().FieldByName(stID)
	fromST := fromSTField.Interface().(float64)
	toST := toSTField.Interface().(float64)
	//update st balance
	fromST -= transfer.Size
	toST += transfer.Size
	fromSTValue := reflect.ValueOf(fromST)
	toSTValue := reflect.ValueOf(toST)
	fromSTField.Set(fromSTValue)
	toSTField.Set(toSTValue)
	//update fiat balance
	fromAccount.Fiat += transfer.Size * transfer.Price
	toAccount.Fiat -= transfer.Size * transfer.Price
	return nil
}

func (s *SmartContract) CreateTransfersInBatch(ctx contractapi.TransactionContextInterface, transferJSONBatchString string) (error) {
	transfers, _ := s.unmarshalTransferBatchString(transferJSONBatchString)
    for _, transfer := range transfers {
		transferJSON, jsonErr := json.Marshal(transfer)
		if jsonErr != nil {
			return jsonErr
		}
		err := ctx.GetStub().PutState(transfer.TransferId, transferJSON)
		if err != nil {
			return fmt.Errorf("failed to put to world state. %v", err)
		}
    }
	return nil
}


func (s *SmartContract) unmarshalTransferBatchString(transferBatchString string) ([]Transfer, error) {
	var transferBatchInterface []map[string]interface{}
    err := json.Unmarshal([]byte(transferBatchString), &transferBatchInterface)
	fmt.Println("Parsed JSON data:")
	if err != nil {
		return nil, fmt.Errorf("Error while unmarshalling transactionBatchString: %w", err)
	}
	var transfers []Transfer
	for _, item := range transferBatchInterface {
		transfer := Transfer{
			FromAddress: item["FromAddress"].(string),
			Price:       item["Price"].(float64),
			ST_ID:       item["ST_ID"].(string),
			Size:        item["Size"].(float64),
			TransferId:  item["TransferId"].(string),
			ToAddress:   item["ToAddress"].(string),
		}
		if transfer.TransferId != "" {
			transfers = append(transfers, transfer)
		}
	}
	return transfers, nil
}


func (s *SmartContract) verifySufficientBalance(ctx contractapi.TransactionContextInterface, 
	accounts map[string]*Account, transfer Transfer) (error) {
	
	// Get source and destination accounts
	fromAccount, fromExists := accounts[transfer.FromAddress]
	toAccount, toExists := accounts[transfer.ToAddress]
	if !fromExists || !toExists {
		return fmt.Errorf("Source or destination account does not exist")
	}

	fromAccountST, _ := fromAccount.getSTBalance(transfer.ST_ID)
	requiredST := transfer.Size
	requiredFiat := requiredST * transfer.Price

	if  fromAccountST < requiredST {
		return fmt.Errorf("Insufficient ST balance in %s account", fromAccount.Address)
	}	

	if toAccount.Fiat < requiredFiat {
		return fmt.Errorf("Insufficient fiat balance in %s account", toAccount.Address)
	}

	return nil
}

func (a Account) getSTBalance(stID string) (float64, error) {
    // Account 구조체를 reflection을 사용하여 탐색
    valueOf := reflect.ValueOf(a)

    // stID에 해당하는 필드를 가져옴
    field := valueOf.FieldByName(stID)

    if !field.IsValid() {
        return 0, fmt.Errorf("Field not found: %s", stID)
    }

    // 필드의 값(잔액)을 float64로 변환
    stBalance := field.Interface().(float64)

    return stBalance, nil
}

//사용 X
func (s *SmartContract) CreateTransfer(ctx contractapi.TransactionContextInterface, 
	transferId string, stId string, fromAddress string, toAddress string, size float64, price float64) error {
	exists, err := s.TransferExists(ctx, transferId)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("the transferId for transfer %s already exists", transferId)
	}

	transfer :=  Transfer {
		FromAddress:   fromAddress,
		Price:         price,
		ST_ID:         stId,
		Size:          size,
		TransferId:    transferId,
		ToAddress:     toAddress,
	}

	transferJSON, err := json.Marshal(transfer)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(transferId, transferJSON)
}

// ReadTransfer returns the transfer stored in the world state with given id.
func (s *SmartContract) ReadTransfer(ctx contractapi.TransactionContextInterface, transferId string) (*Transfer, error) {
	transferJSON, err := ctx.GetStub().GetState(transferId)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if transferJSON == nil {
		return nil, fmt.Errorf("the account %s does not exist", transferId)
	}

	var transfer Transfer
	err = json.Unmarshal(transferJSON, &transfer)
	if err != nil {
		return nil, err
	}

	return &transfer, nil
}


// CreateAccount issues a new account to the world state with given details.
func (s *SmartContract) CreateAccount(ctx contractapi.TransactionContextInterface, address string, fiat float64, st_1 float64) (error) {
	exists, err := s.AccountExists(ctx, address)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("the address for account %s already exists", address)
	}

	account := Account{
		Address: address,
		Fiat:    fiat,
		ST_1:    st_1,
	}

	accountJSON, err := json.Marshal(account)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(address, accountJSON)
}

func (s *SmartContract) CreateAccountTest(ctx contractapi.TransactionContextInterface, jsonString string) (error) {
	var account Account
	err := json.Unmarshal([]byte(jsonString), &account)
	if err != nil {
		return err
	}
	accountJSON, err := json.Marshal(account)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(account.Address, accountJSON)
}

// ReadAccount returns the account stored in the world state with given address.
func (s *SmartContract) ReadAccount(ctx contractapi.TransactionContextInterface, address string) (*Account, error) {
	accountJSON, err := ctx.GetStub().GetState(address)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if accountJSON == nil {
		return nil, fmt.Errorf("the account %s does not exist", address)
	}

	var account Account
	err = json.Unmarshal(accountJSON, &account)
	if err != nil {
		return nil, err
	}

	return &account, nil
}

//State에 Account, Transfer 두 가지 유형이 혼합해서 들어있기 때문에 수정이 필요. 사용할 때 Transfer가 ledger에 있으면 오류 반환.
// GetAllAccounts returns all accounts found in world state
func (s *SmartContract) GetAllAccounts(ctx contractapi.TransactionContextInterface) (map[string]*Account, error) {
	// range query with empty string for startKey and endKey does an
	// open-ended query of all accounts in the chaincode namespace.
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	accounts := make(map[string]*Account)
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var account Account
		err = json.Unmarshal(queryResponse.Value, &account)
		if err != nil {
			return nil, err
		}

		accounts[account.Address] = &account
	}

	return accounts, nil
}

func (s *SmartContract) GetAllTransfers(ctx contractapi.TransactionContextInterface) (map[string]*Transfer, error) {
	// range query with empty string for startKey and endKey does an
	// open-ended query of all accounts in the chaincode namespace.
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	transfers := make(map[string]*Transfer)
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var transfer Transfer
		err = json.Unmarshal(queryResponse.Value, &transfer)
		if err != nil {
			return nil, err
		}

		transfers[transfer.TransferId] = &transfer
	}

	return transfers, nil
}

// UpdateAccount updates an existing account in the world state with provaddressed parameters.
func (s *SmartContract) UpdateAccountByObject(ctx contractapi.TransactionContextInterface, account Account) error {
	exists, err := s.AccountExists(ctx, account.Address)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("the account %s does not exist", account.Address)
	}

	// overwriting original account with new account

	accountJSON, err := json.Marshal(account)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(account.Address, accountJSON)
}

// UpdateAccount updates an existing account in the world state with provaddressed parameters.
func (s *SmartContract) UpdateAccountByParams(ctx contractapi.TransactionContextInterface, address string, fiat float64, st_1 float64) error {
	exists, err := s.AccountExists(ctx, address)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("the account %s does not exist", address)
	}

	// overwriting original account with new account
	account := Account{
		Address:   address,
		Fiat:      fiat,
		ST_1:      st_1,
	}

	accountJSON, err := json.Marshal(account)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(address, accountJSON)
}

// DeleteAccount deletes an given account from the world state.
func (s *SmartContract) DeleteAccount(ctx contractapi.TransactionContextInterface, address string) error {
	exists, err := s.AccountExists(ctx, address)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("the account %s does not exist", address)
	}

	return ctx.GetStub().DelState(address)
}

func (s *SmartContract) DeleteTransfer(ctx contractapi.TransactionContextInterface, transferId string) error {

	return ctx.GetStub().DelState(transferId)
}


// AccountExists returns true when account with given address exists in world state
func (s *SmartContract) AccountExists(ctx contractapi.TransactionContextInterface, address string) (bool, error) {
	accountJSON, err := ctx.GetStub().GetState(address)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}

	return accountJSON != nil, nil
}

// TransferExists returns true when account with given address exists in world state
func (s *SmartContract) TransferExists(ctx contractapi.TransactionContextInterface, transferId string) (bool, error) {
	accountJSON, err := ctx.GetStub().GetState(transferId)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}

	return accountJSON != nil, nil
}