package keystore

import (
	"PhoenixOracle/core/keystore/keys/vrfkey"
	"gorm.io/gorm"
)

type VRFORM interface {
	FirstOrCreateEncryptedSecretVRFKey(k *vrfkey.EncryptedVRFKey) error
	ArchiveEncryptedSecretVRFKey(k *vrfkey.EncryptedVRFKey) error
	DeleteEncryptedSecretVRFKey(k *vrfkey.EncryptedVRFKey) error
	FindEncryptedSecretVRFKeys(where ...vrfkey.EncryptedVRFKey) ([]*vrfkey.EncryptedVRFKey, error)
	FindEncryptedSecretVRFKeysIncludingArchived(where ...vrfkey.EncryptedVRFKey) ([]*vrfkey.EncryptedVRFKey, error)
}

type vrfORM struct {
	db *gorm.DB
}

var _ VRFORM = &vrfORM{}

func NewVRFORM(db *gorm.DB) VRFORM {
	return &vrfORM{
		db: db,
	}
}

func (orm *vrfORM) FirstOrCreateEncryptedSecretVRFKey(k *vrfkey.EncryptedVRFKey) error {
	return orm.db.FirstOrCreate(k).Error
}

func (orm *vrfORM) ArchiveEncryptedSecretVRFKey(k *vrfkey.EncryptedVRFKey) error {
	return orm.db.Delete(k).Error
}

func (orm *vrfORM) DeleteEncryptedSecretVRFKey(k *vrfkey.EncryptedVRFKey) error {
	return orm.db.Unscoped().Delete(k).Error
}

func (orm *vrfORM) FindEncryptedSecretVRFKeys(where ...vrfkey.EncryptedVRFKey) (
	retrieved []*vrfkey.EncryptedVRFKey, err error) {
	var anonWhere []interface{} // Find needs "where" contents coerced to interface{}
	for _, constraint := range where {
		c := constraint
		anonWhere = append(anonWhere, &c)
	}
	return retrieved, orm.db.Find(&retrieved, anonWhere...).Order("created_at DESC, id DESC").Error
}

func (orm *vrfORM) FindEncryptedSecretVRFKeysIncludingArchived(where ...vrfkey.EncryptedVRFKey) (
	retrieved []*vrfkey.EncryptedVRFKey, err error) {
	var anonWhere []interface{} // Find needs "where" contents coerced to interface{}
	for _, constraint := range where {
		c := constraint
		anonWhere = append(anonWhere, &c)
	}
	return retrieved, orm.db.Unscoped().Find(&retrieved, anonWhere...).Order("created_at DESC, id DESC").Error
}
