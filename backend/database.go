package backend

import (
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

// ObjID bson.ObjectId type alias for ease of use
type ObjID = bson.ObjectId

// Collection mgo.Collection alias
type Collection = mgo.Collection

// DialInfo mgo.DialInfo alias
type DialInfo = mgo.DialInfo

// Database mongodb convenience wrapper
type Database struct {
	Session *mgo.Session
	DB      *mgo.Database
	Users   *Collection
	Writs   *Collection
}

// Open start a mgo session and get a DB
func (db *Database) Open(dailinfo *DialInfo, dbName string) {
	session, err := mgo.DialWithInfo(dailinfo)
	if err != nil {
		panic(err)
	}
	// session.SetMode(mgo.Monotonic, true)
	session.SetSafe(&mgo.Safe{})
	db.Session = session
	db.DB = db.Session.DB(dbName)
	db.Users = db.Coll("users")
	db.Writs = db.Coll("writs")
}

// Close the mgo.Session
func (db *Database) Close() {
	db.Session.Close()
}

// Coll get a mgo.Collection
func (db *Database) Coll(collName string) *Collection {
	return db.DB.C(collName)
}

// MakeID generate a new bson.ObjectId
func MakeID() ObjID {
	return bson.NewObjectId()
}
