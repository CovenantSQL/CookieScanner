package covenant

import (
	"bytes"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"reflect"
	"testing"
	"time"
)

var (
	myInt1 = MyInt(1)
	myInt2 = MyInt(2)
	tm = time.Now()
	v1 = HeaderTest{
		Version:   110,
		TestName:  "31231",
		TestArray: []byte{0x11, 0x22},
		Producer:  "rewqrwe",
		GenesisHash: []hash.Hash{
			{0x10},
			{0x20},
		},
		ParentHash: []*hash.Hash{
			{0x10},
			{0x20},
		},
		MerkleRoot: &[]*hash.Hash{
			{0x10},
			{0x20},
		},
		Timestamp: tm,
		S: Struct{
			Which: map[string]*MyInt{"s": &myInt2, "ss": &myInt1},
			Other: []byte{'1', '0', 'a'},
			Nums:  [8]float64{1.1, 222.0, 111},
		},
		xx: 0,
	}
	v11 = HeaderTest{
		Version:   110,
		TestName:  "31231",
		TestArray: []byte{0x11, 0x22},
		Producer:  "rewqrwe",
		GenesisHash: []hash.Hash{
			{0x10},
			{0x20},
		},
		ParentHash: []*hash.Hash{
			{0x10},
			{0x20},
		},
		MerkleRoot: &[]*hash.Hash{
			{0x10},
			{0x20},
		},
		Timestamp: tm,
		S: Struct{
			Which: map[string]*MyInt{"s": &myInt2, "ss": &myInt1},
			Other: []byte{'1', '0', 'a'},
			Nums:  [8]float64{1.1, 222.0, 111},
		},
		xx: 0,
	}
	v2 = HeaderTest2{
		Version2:   110,
		TestName2:  "31231",
		TestArray: []byte{0x11, 0x22},
		Producer2:  "rewqrwe",
		GenesisHash2: []hash.Hash{
			{0x10},
			{0x20},
		},
		ParentHash2: []*hash.Hash{
			{0x10},
			{0x20},
		},
		MerkleRoot2: &[]*hash.Hash{
			{0x10},
			{0x20},
		},
		S: Struct{
			Which: map[string]*MyInt{"ss": &myInt1, "s": &myInt2},
			Other: []byte{'1', '0', 'a'},
			Nums:  [8]float64{1.1, 222.0, 111},
		},
		Timestamp2: tm,
		xx:         1,
	}
)
// test different type and member name but same data type and content hash identical
func TestMarshalHashAccountStable2(t *testing.T) {
	bts1, err := v1.MarshalHash()
	if err != nil {
		t.Fatal(err)
	}
	bts2, err := v2.MarshalHash()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bts1, bts2) {
		t.Fatal("hash not stable")
	}
}

func BenchmarkCompare(b *testing.B) {
	b.Run("benchmark reflect", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if !reflect.DeepEqual(v1, v11) {
				b.Fatal("should be equal")
			}
		}
	})

	b.Run("benchmark hsp", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			bts1, _ := v1.MarshalHash()
			bts2, _ := v11.MarshalHash()
			if !bytes.Equal(bts1, bts2) {
				b.Fatal("hash not stable")
			}
		}
	})

	b.Run("benchmark hsp 1 cached", func(b *testing.B) {
		bts1, _ := v1.MarshalHash()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			bts2, _ := v11.MarshalHash()
			if !bytes.Equal(bts1, bts2) {
				b.Fatal("hash not stable")
			}
		}
	})

	b.Run("benchmark hsp both cached", func(b *testing.B) {
		bts1, _ := v1.MarshalHash()
		bts2, _ := v11.MarshalHash()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if !bytes.Equal(bts1, bts2) {
				b.Fatal("hash not stable")
			}
		}
	})
}

// test different type and member name but same data type and content hash identical
func TestMarshalHashAccountStable3(t *testing.T) {
	p1 := Person1{
		Name:       "Auxten",
		Address:    "@CovenantSQL.io",
		Age:        70,
		Map:         map[string]int{"ss": 2, "s": 1, "sss": 3},
		unexported: false,
	}
	p2 := Person2{
		Name:       "Auxten",
		Address:    "@CovenantSQL.io",
		Age:        70,
		Map222:      map[string]int{"ss": 2, "s": 1, "sss": 3},
		unexported: true,
	}
	p3 := Person2{
		Name:       "Auxten",
		Address:    "@CovenantSQL.io",
		Age:        70,
		Map222:      map[string]int{"ss": 2, "s": 1, "sss333333": 3},
		unexported: true,
	}
	bts1, err := p1.MarshalHash()
	if err != nil {
		t.Fatal(err)
	}
	bts2, err := p2.MarshalHash()
	if err != nil {
		t.Fatal(err)
	}
	bts3, err := p3.MarshalHash()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(bts1, []byte("Address")) {
		t.Fatal("should not contain any key")
	}
	if !bytes.Equal(bts1, bts2) {
		t.Fatal("hash not stable")
	}
	if bytes.Equal(bts1, bts3) {
		t.Fatal("hash should not equal")
	}
}
