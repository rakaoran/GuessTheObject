package configs

// argon2.IDKey([]byte(password), salt, 2, 1024*4, 2, 32)
var Argon2id = struct {
	Time    uint32
	Memory  uint32
	Threads uint8
	KeyLen  uint32
}{
	Time:    1,
	Memory:  1024 * 64,
	Threads: 2,
	KeyLen:  32,
}
