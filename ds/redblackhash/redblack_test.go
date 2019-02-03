package redblackhash

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestTree_Marshal(t *testing.T) {
	var t1, t2 Tree

	// Generate 1000 random values to insert
	for i := 0; i < 1000; i++ {
		var key Key
		rand.Read(key[:])
		t1.Put(&key)
	}

	// Marshal tree
	var wr bytes.Buffer
	err := t1.Marshal(&wr)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	buf := wr.Bytes()
	rd := bytes.NewBuffer(buf)

	// Unmarshal tree
	err = t2.Unmarshal(rd)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if !compare(t1.Root, t2.Root) {
		t.Error("trees are not equal")
		t.FailNow()
	}
}

func compare(n1, n2 *Node) bool {
	return n1.Key.Compare(&n2.Key) == 0 &&
		(n1.Left == nil || compare(n1.Left, n2.Left)) &&
		(n1.Right == nil || compare(n1.Right, n2.Right))
}
