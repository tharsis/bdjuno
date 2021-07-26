package database

import (
	"fmt"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	dbtypes "github.com/forbole/bdjuno/database/types"
	dbutils "github.com/forbole/bdjuno/database/utils"
	"github.com/forbole/bdjuno/types"
)

// SaveAccounts saves the given accounts inside the database
func (db *Db) SaveAccounts(accounts []types.Account) error {
	paramsNumber := 1
	slices := dbutils.SplitAccounts(accounts, paramsNumber)

	for _, accounts := range slices {
		if len(accounts) == 0 {
			continue
		}

		// Store up-to-date data
		err := db.saveAccounts(paramsNumber, accounts)
		if err != nil {
			return fmt.Errorf("error while storing accounts: %s", err)
		}
	}

	return nil
}

func (db *Db) saveAccounts(paramsNumber int, accounts []types.Account) error {
	if len(accounts) == 0 {
		return nil
	}
	stmt := `INSERT INTO account (address,details) VALUES `
	var params []interface{}
	patchSize:=65535 
	patchCount := 0
	
	for i, account := range accounts {
		ai := patchCount * 2
		stmt += fmt.Sprintf("($%d,$%d),", ai+1,ai+2)
		protoContent, ok := account.Details.(authtypes.AccountI)
		if !ok {
			return fmt.Errorf("invalid proposal content types: %T", account.Details)
		}

		anyContent, err := codectypes.NewAnyWithValue(protoContent)
		if err != nil {
			return err
		}

		contentBz, err := db.EncodingConfig.Marshaler.MarshalJSON(anyContent)
		if err != nil {
			return err
		}

		contentBzstring:=string(contentBz)

		params = append(params, account.Address,contentBzstring)
		if (patchCount==patchSize || i==(len(accounts)-1)){
			stmt = stmt[:len(stmt)-1]
			stmt += " ON CONFLICT (address) DO UPDATE SET details = excluded.details"
			_, err := db.Sql.Exec(stmt, params...)
			if err!=nil{
				return err
			}

			//Initialise
			stmt = `INSERT INTO account (address,details) VALUES `
			patchCount=0
			params = make([]interface{}, 0)

		}
		patchCount++
	}
	return nil
}

// GetAccounts returns all the accounts that are currently stored inside the database.
func (db *Db) GetAccounts() ([]types.Account, error) {
	var rows []dbtypes.AccountRow
	err := db.Sqlx.Select(&rows, `SELECT address,details FROM account`)
	if err!=nil{
		return nil,err
	}

	returnRows:=make([]types.Account,len(rows))
	for i,row:=range rows {
		b := []byte(row.Details)
		
		if len(b)==0{
			returnRows[i]=types.NewAccount(row.Address,nil)
		}else{
			//var inter interface{}
			var a codectypes.Any
			db.EncodingConfig.Marshaler.MustUnmarshalJSON(b,&a)
			if &a==nil{
				return nil,fmt.Errorf("UnMarshalJson return nil ")
			}
			//err=json.Unmarshal(b,inter)

			if err!=nil{
				return nil,err
			}
		
			//accI,ok:=(authtypes.AccountI)(a)
			accI:=a.GetCachedValue()
			
			if accI==nil{
				return nil,fmt.Errorf("CachedValue return nil")
			}else{
			returnRows[i]=types.NewAccount(row.Address,accI.(authtypes.AccountI))
			}
		}
	}
	return returnRows, err
}
