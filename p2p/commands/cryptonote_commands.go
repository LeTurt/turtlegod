package commands

import (
	"github.com/leturt/turtlegod/p2p/parser"
	"github.com/leturt/turtlegod/p2p/datamodel"
)

func parse2002(data []byte) {
	kvs, _ := parser.ReadSection(data)
	txs := kvs["txs"].([]interface{})
	data = txs[1].([]uint8)
	ParseTransaction(data)
}

func ParseTransaction(data []byte) {
	transaction := datamodel.Transaction{}
	version, bytesRead := parser.UnpackCNVarIntUint8(data)
	transaction.Version = version
	data = data[bytesRead:]

	unlockTime, bytesRead := parser.UnpackCNVarIntUint64(data)
	transaction.UnlockTime = unlockTime
	data = data[bytesRead:]

	inputCount, bytesRead := parser.UnpackCNVarIntUint64(data)
	data = data[bytesRead:]

	transaction.TxIns = make([]datamodel.TxInput, inputCount, inputCount)
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

		txIn := datamodel.TxInput{}
		//create a "slice" just big enough to hold all output indices to read
		outIndices := make([]uint32, outputCount, outputCount)
		keyImage := make([]uint8, 32, 32)
		txIn.Key = datamodel.TxKeyInput{amount, outIndices, keyImage}

		for o := 0 ; o < int(outputCount) ; o++ {
			outputIndex, bytesRead := parser.UnpackCNVarIntUint32(data)
			data = data[bytesRead:]
			txIn.Key.TxOutIndices[o] = outputIndex
		}
		//		keyImage := []uint8(datamodel[0:32])
		copy(keyImage, []uint8(data[0:32]))
		data = data[32:]
		transaction.TxIns[i] = txIn
	}

	//TXOUTPUT list read start (from transaction prefix)
	outputCount, bytesRead := parser.UnpackCNVarIntUint64(data)
	data = data[bytesRead:]
	transaction.TxOuts = make([]datamodel.TxOutput, outputCount, outputCount)
	for i := 0 ; i < int(outputCount) ; i++ {
		amount, bytesRead := parser.UnpackCNVarIntUint64(data)
		data = data[bytesRead:]
		typeTag := uint8(data[0])
		data = data[1:]
		if typeTag != 0x02 {
			panic("Expecting only tx_to_key type tags in transaction input.")
		}
		key := make([]uint8, 32, 32)
		copy(key, []uint8(data[0:32]))
		txOut := datamodel.TxOutput{amount, key}
		data = data[32:]
		transaction.TxOuts[i] = txOut
	}

	//TXEXTRA read
	extraSize, bytesRead := parser.UnpackCNVarIntUint64(data)
	data = data[bytesRead:]
	transaction.TxExtra = make([]byte, extraSize, extraSize)
	copy(transaction.TxExtra, data[0:extraSize])
	data = data[extraSize:]


	transaction.Signatures = make([][][]uint8, len(transaction.TxIns))
	for idx, txIn := range transaction.TxIns {
		count := len(txIn.Key.TxOutIndices)
		transaction.Signatures[idx] = make([][]uint8, count)
		println("processing tx idx ", idx)
		for s := 0 ; s < count ; s++ {
			signature := make([]uint8, 64, 64)
			copy(signature, []uint8(data[0:64]))
			data = data[64:]
			//for each output index, need a sig
			transaction.Signatures[idx][s] = signature
		}
	}
}