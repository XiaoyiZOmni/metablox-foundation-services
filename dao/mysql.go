package dao

import (
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/metabloxDID/models"
	logger "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var SqlDB *sqlx.DB

func InitSql() error {
	var err error

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		viper.GetString("mysql.user"),
		viper.GetString("mysql.password"),
		viper.GetString("mysql.host"),
		viper.GetString("mysql.port"),
		viper.GetString("mysql.dbname"),
	)

	SqlDB, err = sqlx.Open("mysql", dsn)
	if err != nil {
		logger.Error("Failed to open database: " + err.Error())
		return err
	}

	//Set the maximum number of database connections

	SqlDB.SetConnMaxLifetime(100)

	//Set the maximum number of idle connections on the database

	SqlDB.SetMaxIdleConns(10)

	//Verify connection

	if err := SqlDB.Ping(); err != nil {
		logger.Error("open database fail: ", err)
		return err
	}
	logger.Info("connect success")
	return nil
}

func Close() {
	SqlDB.Close()
}

func UploadWifiAccessVC(vc models.VerifiableCredential) (int, error) {
	tx, err := SqlDB.Beginx()
	if err != nil {
		return 0, err
	}

	sqlStr := "insert into Credentials (Type, Issuer, IssuanceDate, ExpirationDate, Description, Revoked) values (:Type, :Issuer, :IssuanceDate, :ExpirationDate, :Description, 0)"
	result, err := tx.NamedExec(sqlStr, vc)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	newID, _ := result.LastInsertId()
	sqlStr = "insert into WifiAccessInfo (CredentialID, PlaceholderParameter) values (?,?)"
	wifiAccessInfo := vc.CredentialSubject.(models.WifiAccessInfo)
	_, err = tx.Exec(sqlStr, newID, wifiAccessInfo.PlaceholderParameter)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	err = tx.Commit()
	if err != nil {
		return 0, err
	}
	return int(newID), nil
}

func UploadMiningLicenseVC(vc models.VerifiableCredential) (int, error) {
	tx, err := SqlDB.Beginx()
	if err != nil {
		return 0, err
	}

	sqlStr := "insert into Credentials (Type, Issuer, IssuanceDate, ExpirationDate, Description, Revoked) values (:Type, :Issuer, :IssuanceDate, :ExpirationDate, :Description, 0)"
	result, err := tx.NamedExec(sqlStr, vc)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	newID, _ := result.LastInsertId()
	sqlStr = "insert into MiningLicenseInfo (CredentialID, PlaceholderParameter2) values (?,?)"
	miningLicenseInfo := vc.CredentialSubject.(models.MiningLicenseInfo)
	_, err = tx.Exec(sqlStr, newID, miningLicenseInfo.PlaceholderParameter2)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	err = tx.Commit()
	if err != nil {
		return 0, err
	}
	return int(newID), nil
}

func UpdateVCExpirationDate(id, expirationDate string) error {
	sqlStr := "update Credentials set ExpirationDate = ? where ID = ?"
	_, err := SqlDB.Exec(sqlStr, expirationDate, id)
	if err != nil {
		return err
	}
	return nil
}

func RevokeVC(id string) error {
	sqlStr := "update Credentials set Revoked = 1 where ID = ?"
	_, err := SqlDB.Exec(sqlStr, id)
	if err != nil {
		return err
	}
	return nil
}

func GetCredentialStatusByID(id string) (bool, error) {
	sqlStr := "select Revoked from Credentials where ID = ?"
	var revoked bool
	err := SqlDB.Get(&revoked, sqlStr, id)
	if err != nil {
		return false, err
	}

	return revoked, nil
}
