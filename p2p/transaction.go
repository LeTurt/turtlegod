package p2p

import "github.com/leturt/turtlegod/p2p/parser"

//a single field struct just to give it a name
type PublicKey struct {
	key []uint8  //32 bytes
}

type TxBaseInput struct {
	blockIndex uint32
}

type TxKeyInput struct {
	amount uint64
	txOutIndices []uint32
	keyImage []uint8 //32 bytes
}

type TxInput struct {
	isKey bool //if true, this is key input type, and ignore base. otherwise the opposite
	base TxBaseInput
	key TxKeyInput
}

type TxOutput struct {
	amount uint64
	target []uint8 //someones public key
}

type Transaction struct {
	//following 5 are "transaction prefix" in CN code
	version uint8
	unlockTime uint64
	txIns []TxInput    //txin
	txOuts []TxOutput
	txExtra []byte
	//signatures is in "transaction" in CN code, which extends "transaction prefix"
	signatures [][][]uint8
}

func parseTransaction(data []byte) {
	transaction := Transaction{}
	version, bytesRead := parser.UnpackCNVarIntUint8(data)
	transaction.version = version
	data = data[bytesRead:]

	unlockTime, bytesRead := parser.UnpackCNVarIntUint64(data)
	transaction.unlockTime = unlockTime
	data = data[bytesRead:]

	inputCount, bytesRead := parser.UnpackCNVarIntUint64(data)
	data = data[bytesRead:]

	transaction.txIns = make([]TxInput, inputCount, inputCount)
	//TXINPUT list read start (from transaction prefix)
	//assume its not over max int size..
	for i := 0 ; i < int(inputCount) ; i++ {
		typeTag := uint8(data[0])
		if typeTag != 0x02 {
			panic("Expecting only KeyInput type tags in transaction input.")
		}
		data = data[1:]

		amount, bytesRead := parser.UnpackCNVarIntUint64(data)
		data = data[bytesRead:]
		outputCount, bytesRead := parser.UnpackCNVarIntUint64(data)
		data = data[bytesRead:]

		txIn := TxInput{}
		//create a "slice" just big enough to hold all output indices to read
		outIndices := make([]uint32, outputCount, outputCount)
		keyImage := make([]uint8, 32, 32)
		txIn.key = TxKeyInput{amount, outIndices, keyImage}

		for o := 0 ; o < int(outputCount) ; o++ {
			outputIndex, bytesRead := parser.UnpackCNVarIntUint32(data)
			data = data[bytesRead:]
			txIn.key.txOutIndices[o] = outputIndex
		}
//		keyImage := []uint8(data[0:32])
		copy(keyImage, []uint8(data[0:32]))
		data = data[32:]
		transaction.txIns[i] = txIn
	}

	//TXOUTPUT list read start (from transaction prefix)
	outputCount, bytesRead := parser.UnpackCNVarIntUint64(data)
	data = data[bytesRead:]
	transaction.txOuts = make([]TxOutput, outputCount, outputCount)
	for i := 0 ; i < int(outputCount) ; i++ {
		typeTag := uint8(data[0])
		if typeTag != 0x02 {
			panic("Expecting only KeyInput type tags in transaction input.")
		}
		amount, bytesRead := parser.UnpackCNVarIntUint64(data)
		data = data[bytesRead:]
		key := make([]uint8, 32, 32)
		copy(key, []uint8(data[0:32]))
		txOut := TxOutput{amount, key}
		data = data[32:]
		transaction.txOuts[i] = txOut
	}

	//TXEXTRA read
	extraSize, bytesRead := parser.UnpackCNVarIntUint64(data)
	data = data[bytesRead:]
	transaction.txExtra = make([]byte, extraSize, extraSize)
	copy(transaction.txExtra, data[0:extraSize])
	data = data[extraSize:]

	for idx, txIn := range transaction.txIns {
		println("processing tx idx ", idx)
		for s := 0 ; s < len(txIn.key.txOutIndices) ; s++ {
			signature := make([]uint8, 64, 64)
			copy(signature, []uint8(data[0:64]))
			data = data[64:]
			//for each output index, need a sig
			transaction.signatures[idx][s] = signature
		}
	}
}