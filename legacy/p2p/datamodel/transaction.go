package datamodel

//a single field struct just to give it a name
type PublicKey struct {
	Key []uint8  //32 bytes
}

type TxBaseInput struct {
	BlockIndex uint32
}

type TxKeyInput struct {
	Amount uint64
	TxOutIndices []uint32
	KeyImage []uint8 //32 bytes
}

type TxInput struct {
	IsKey bool //if true, this is key input type, and ignore base. otherwise the opposite
	Base TxBaseInput
	Key TxKeyInput
}

type TxOutput struct {
	Amount uint64
	Target []uint8 //someones public key
}

type Transaction struct {
	//following 5 are "transaction prefix" in CN code
	Version uint8
	UnlockTime uint64
	TxIns []TxInput    //txin
	TxOuts []TxOutput
	TxExtra []byte
	//signatures is in "transaction" in CN code, which extends "transaction prefix"
	Signatures [][][]uint8
}

