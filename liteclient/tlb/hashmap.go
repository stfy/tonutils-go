package tlb

import (
	"bytes"
	"encoding/hex"
	"math"

	"github.com/xssnick/tonutils-go/tvm/cell"
)

type Hashmap struct {
	storage map[string]*cell.Cell
}

type HashmapKV struct {
	Key   string
	Value *cell.Cell
}

func (h *Hashmap) LoadFromCell(keySz int, loader *cell.LoadCell) error {
	h.storage = map[string]*cell.Cell{}

	err := h.mapInner(keySz, keySz, loader, cell.BeginCell())
	if err != nil {
		return err
	}

	return nil
}

func (h *Hashmap) Get(key *cell.Cell) *cell.Cell {
	return h.storage[hex.EncodeToString(key.Hash())]
}

func (h *Hashmap) All() []HashmapKV {
	all := make([]HashmapKV, 0, len(h.storage))
	for k, v := range h.storage {
		all = append(all, HashmapKV{
			Key:   k,
			Value: v,
		})
	}

	return all
}

func (h *Hashmap) mapInner(keySz, leftKeySz int, loader *cell.LoadCell, keyPrefix *cell.Builder) error {
	var err error
	var sz int

	sz, keyPrefix, err = loadLabel(leftKeySz, loader, keyPrefix)
	if err != nil {
		return err
	}

	key := keyPrefix.EndCell().BeginParse()

	// until key size is not equals we go deeper
	if key.BitsLeft() < keySz {
		// 0 bit branch
		left, err := loader.LoadRef()
		if err != nil {
			return nil
		}
		err = h.mapInner(keySz, leftKeySz-(1+sz), left, keyPrefix.Copy().MustStoreUInt(0, 1))
		if err != nil {
			return err
		}

		// 1 bit branch
		right, err := loader.LoadRef()
		if err != nil {
			return err
		}
		err = h.mapInner(keySz, leftKeySz-(1+sz), right, keyPrefix.Copy().MustStoreUInt(1, 1))
		if err != nil {
			return err
		}

		return nil
	}

	// add node to map
	h.storage[hex.EncodeToString(keyPrefix.EndCell().Hash())] = loader.MustToCell()

	return nil
}

func loadLabel(sz int, loader *cell.LoadCell, key *cell.Builder) (int, *cell.Builder, error) {
	first, err := loader.LoadUInt(1)
	if err != nil {
		return 0, nil, err
	}

	// hml_short$0
	if first == 0 {
		// Unary, while 1, add to ln
		ln := 0
		for {
			bit, err := loader.LoadUInt(1)
			if err != nil {
				return 0, nil, err
			}

			if bit == 0 {
				break
			}
			ln++
		}

		keyBits, err := loader.LoadSlice(ln)
		if err != nil {
			return 0, nil, err
		}

		// add bits to key
		err = key.StoreSlice(keyBits, ln)
		if err != nil {
			return 0, nil, err
		}

		return ln, key, nil
	}

	second, err := loader.LoadUInt(1)
	if err != nil {
		return 0, nil, err
	}

	// hml_long$10
	if second == 0 {
		bitsLen := int(math.Ceil(math.Log2(float64(sz + 1))))

		ln, err := loader.LoadUInt(bitsLen)
		if err != nil {
			return 0, nil, err
		}

		keyBits, err := loader.LoadSlice(int(ln))
		if err != nil {
			return 0, nil, err
		}

		// add bits to key
		err = key.StoreSlice(keyBits, int(ln))
		if err != nil {
			return 0, nil, err
		}

		return int(ln), key, nil
	}

	// hml_same$11
	bitType, err := loader.LoadUInt(1)
	if err != nil {
		return 0, nil, err
	}

	bitsLen := int(math.Ceil(math.Log2(float64(sz + 1))))

	ln, err := loader.LoadUInt(bitsLen)
	if err != nil {
		return 0, nil, err
	}

	var toStore []byte
	if bitType == 1 {
		// N of ones
		toStore = bytes.Repeat([]byte{0xFF}, 1+(int(ln)/8))
	} else {
		// N of zeroes
		toStore = bytes.Repeat([]byte{0x00}, 1+(int(ln)/8))
	}

	err = key.StoreSlice(toStore, int(ln))
	if err != nil {
		return 0, nil, err
	}

	return int(ln), key, nil
}
