package trace

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"sync"
	"unsafe"
)

const (
	idSize  = aes.BlockSize / 2 // 64 bits
	keySize = aes.BlockSize     // 128 bits
)

var (
	ctr []byte
	n   int
	b   []byte
	c   cipher.Block
	m   sync.Mutex
)

func init() {
	buf := make([]byte, keySize+aes.BlockSize)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		panic(err) // /dev/urandom had better work
	}
	c, err = aes.NewCipher(buf[:keySize])
	if err != nil {
		panic(err) // AES had better work
	}
	n = aes.BlockSize
	ctr = buf[keySize:]
	b = make([]byte, aes.BlockSize)
}

// generateID returns a randomly-generated 64-bit ID. This function is
// thread-safe.  IDs are produced by consuming an AES-CTR-128 keystream in
// 64-bit chunks. The AES key is randomly generated on initialization, as is the
// counter's initial state. On machines with AES-NI support, ID generation takes
// ~40ns and generates no garbage.
func generateID() uint64 {
	m.Lock()
	if n == aes.BlockSize {
		c.Encrypt(b, ctr)
		for i := aes.BlockSize - 1; i >= 0; i-- { // increment ctr
			ctr[i]++
			if ctr[i] != 0 {
				break
			}
		}
		n = 0
	}
	id := *(*uint64)(unsafe.Pointer(&b[n])) // zero-copy b/c we're arch-neutral
	n += idSize
	m.Unlock()
	return id
}
