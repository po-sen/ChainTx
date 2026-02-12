package walletkeys

// This implementation is adapted from golang.org/x/crypto/ripemd160
// (Copyright 2010 The Go Authors, BSD-style license).

import "math/bits"

const (
	ripemd160Size      = 20
	ripemd160BlockSize = 64

	ripemd160S0 = 0x67452301
	ripemd160S1 = 0xefcdab89
	ripemd160S2 = 0x98badcfe
	ripemd160S3 = 0x10325476
	ripemd160S4 = 0xc3d2e1f0
)

type ripemd160Digest struct {
	s  [5]uint32
	x  [ripemd160BlockSize]byte
	nx int
	tc uint64
}

func ripemd160Sum(input []byte) [ripemd160Size]byte {
	digest := newRIPEMD160Digest()
	_, _ = digest.Write(input)
	return digest.checkSum()
}

func newRIPEMD160Digest() ripemd160Digest {
	digest := ripemd160Digest{}
	digest.Reset()
	return digest
}

func (d *ripemd160Digest) Reset() {
	d.s[0], d.s[1], d.s[2], d.s[3], d.s[4] = ripemd160S0, ripemd160S1, ripemd160S2, ripemd160S3, ripemd160S4
	d.nx = 0
	d.tc = 0
}

func (d *ripemd160Digest) Write(p []byte) (int, error) {
	nn := len(p)
	d.tc += uint64(nn)

	if d.nx > 0 {
		n := len(p)
		if n > ripemd160BlockSize-d.nx {
			n = ripemd160BlockSize - d.nx
		}
		for i := 0; i < n; i++ {
			d.x[d.nx+i] = p[i]
		}
		d.nx += n
		if d.nx == ripemd160BlockSize {
			ripemd160Block(d, d.x[0:])
			d.nx = 0
		}
		p = p[n:]
	}

	n := ripemd160Block(d, p)
	p = p[n:]
	if len(p) > 0 {
		d.nx = copy(d.x[:], p)
	}

	return nn, nil
}

func (d0 *ripemd160Digest) checkSum() [ripemd160Size]byte {
	d := *d0

	tc := d.tc
	var tmp [64]byte
	tmp[0] = 0x80
	if tc%64 < 56 {
		_, _ = d.Write(tmp[0 : 56-tc%64])
	} else {
		_, _ = d.Write(tmp[0 : 64+56-tc%64])
	}

	tc <<= 3
	for i := uint(0); i < 8; i++ {
		tmp[i] = byte(tc >> (8 * i))
	}
	_, _ = d.Write(tmp[0:8])

	if d.nx != 0 {
		panic("ripemd160 digest buffer not fully consumed")
	}

	var out [ripemd160Size]byte
	for i, s := range d.s {
		out[i*4] = byte(s)
		out[i*4+1] = byte(s >> 8)
		out[i*4+2] = byte(s >> 16)
		out[i*4+3] = byte(s >> 24)
	}

	return out
}

var ripemd160N = [80]uint{
	0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
	7, 4, 13, 1, 10, 6, 15, 3, 12, 0, 9, 5, 2, 14, 11, 8,
	3, 10, 14, 4, 9, 15, 8, 1, 2, 7, 0, 6, 13, 11, 5, 12,
	1, 9, 11, 10, 0, 8, 12, 4, 13, 3, 7, 15, 14, 5, 6, 2,
	4, 0, 5, 9, 7, 12, 2, 10, 14, 1, 3, 8, 11, 6, 15, 13,
}

var ripemd160R = [80]uint{
	11, 14, 15, 12, 5, 8, 7, 9, 11, 13, 14, 15, 6, 7, 9, 8,
	7, 6, 8, 13, 11, 9, 7, 15, 7, 12, 15, 9, 11, 7, 13, 12,
	11, 13, 6, 7, 14, 9, 13, 15, 14, 8, 13, 6, 5, 12, 7, 5,
	11, 12, 14, 15, 14, 15, 9, 8, 9, 14, 5, 6, 8, 6, 5, 12,
	9, 15, 5, 11, 6, 8, 13, 12, 5, 12, 13, 14, 11, 8, 5, 6,
}

var ripemd160NPrime = [80]uint{
	5, 14, 7, 0, 9, 2, 11, 4, 13, 6, 15, 8, 1, 10, 3, 12,
	6, 11, 3, 7, 0, 13, 5, 10, 14, 15, 8, 12, 4, 9, 1, 2,
	15, 5, 1, 3, 7, 14, 6, 9, 11, 8, 12, 2, 10, 0, 4, 13,
	8, 6, 4, 1, 3, 11, 15, 0, 5, 12, 2, 13, 9, 7, 10, 14,
	12, 15, 10, 4, 1, 5, 8, 7, 6, 2, 13, 14, 0, 3, 9, 11,
}

var ripemd160RPrime = [80]uint{
	8, 9, 9, 11, 13, 15, 15, 5, 7, 7, 8, 11, 14, 14, 12, 6,
	9, 13, 15, 7, 12, 8, 9, 11, 7, 7, 12, 7, 6, 15, 13, 11,
	9, 7, 15, 11, 8, 6, 6, 14, 12, 13, 5, 14, 13, 13, 7, 5,
	15, 5, 8, 11, 14, 14, 6, 14, 6, 9, 12, 9, 12, 5, 15, 8,
	8, 5, 12, 9, 12, 5, 14, 6, 8, 13, 6, 5, 15, 13, 11, 11,
}

func ripemd160Block(digest *ripemd160Digest, p []byte) int {
	n := 0
	var x [16]uint32

	for len(p) >= ripemd160BlockSize {
		a, b, c, d, e := digest.s[0], digest.s[1], digest.s[2], digest.s[3], digest.s[4]
		aa, bb, cc, dd, ee := a, b, c, d, e

		j := 0
		for i := 0; i < 16; i++ {
			x[i] = uint32(p[j]) | uint32(p[j+1])<<8 | uint32(p[j+2])<<16 | uint32(p[j+3])<<24
			j += 4
		}

		i := 0
		for i < 16 {
			alpha := a + (b ^ c ^ d) + x[ripemd160N[i]]
			alpha = bits.RotateLeft32(alpha, int(ripemd160R[i])) + e
			beta := bits.RotateLeft32(c, 10)
			a, b, c, d, e = e, alpha, b, beta, d

			alpha = aa + (bb ^ (cc | ^dd)) + x[ripemd160NPrime[i]] + 0x50a28be6
			alpha = bits.RotateLeft32(alpha, int(ripemd160RPrime[i])) + ee
			beta = bits.RotateLeft32(cc, 10)
			aa, bb, cc, dd, ee = ee, alpha, bb, beta, dd

			i++
		}

		for i < 32 {
			alpha := a + (b&c | ^b&d) + x[ripemd160N[i]] + 0x5a827999
			alpha = bits.RotateLeft32(alpha, int(ripemd160R[i])) + e
			beta := bits.RotateLeft32(c, 10)
			a, b, c, d, e = e, alpha, b, beta, d

			alpha = aa + (bb&dd | cc&^dd) + x[ripemd160NPrime[i]] + 0x5c4dd124
			alpha = bits.RotateLeft32(alpha, int(ripemd160RPrime[i])) + ee
			beta = bits.RotateLeft32(cc, 10)
			aa, bb, cc, dd, ee = ee, alpha, bb, beta, dd

			i++
		}

		for i < 48 {
			alpha := a + (b | ^c ^ d) + x[ripemd160N[i]] + 0x6ed9eba1
			alpha = bits.RotateLeft32(alpha, int(ripemd160R[i])) + e
			beta := bits.RotateLeft32(c, 10)
			a, b, c, d, e = e, alpha, b, beta, d

			alpha = aa + (bb | ^cc ^ dd) + x[ripemd160NPrime[i]] + 0x6d703ef3
			alpha = bits.RotateLeft32(alpha, int(ripemd160RPrime[i])) + ee
			beta = bits.RotateLeft32(cc, 10)
			aa, bb, cc, dd, ee = ee, alpha, bb, beta, dd

			i++
		}

		for i < 64 {
			alpha := a + (b&d | c&^d) + x[ripemd160N[i]] + 0x8f1bbcdc
			alpha = bits.RotateLeft32(alpha, int(ripemd160R[i])) + e
			beta := bits.RotateLeft32(c, 10)
			a, b, c, d, e = e, alpha, b, beta, d

			alpha = aa + (bb&cc | ^bb&dd) + x[ripemd160NPrime[i]] + 0x7a6d76e9
			alpha = bits.RotateLeft32(alpha, int(ripemd160RPrime[i])) + ee
			beta = bits.RotateLeft32(cc, 10)
			aa, bb, cc, dd, ee = ee, alpha, bb, beta, dd

			i++
		}

		for i < 80 {
			alpha := a + (b ^ (c | ^d)) + x[ripemd160N[i]] + 0xa953fd4e
			alpha = bits.RotateLeft32(alpha, int(ripemd160R[i])) + e
			beta := bits.RotateLeft32(c, 10)
			a, b, c, d, e = e, alpha, b, beta, d

			alpha = aa + (bb ^ cc ^ dd) + x[ripemd160NPrime[i]]
			alpha = bits.RotateLeft32(alpha, int(ripemd160RPrime[i])) + ee
			beta = bits.RotateLeft32(cc, 10)
			aa, bb, cc, dd, ee = ee, alpha, bb, beta, dd

			i++
		}

		dd += c + digest.s[1]
		digest.s[1] = digest.s[2] + d + ee
		digest.s[2] = digest.s[3] + e + aa
		digest.s[3] = digest.s[4] + a + bb
		digest.s[4] = digest.s[0] + b + cc
		digest.s[0] = dd

		p = p[ripemd160BlockSize:]
		n += ripemd160BlockSize
	}

	return n
}
