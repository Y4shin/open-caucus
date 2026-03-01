package votefuzz

import (
	"hash/fnv"
	"math/rand"
)

func DeriveSeed(parent uint64, label string, index int) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(label))
	labelHash := h.Sum64()

	mixed := parent + 0x9e3779b97f4a7c15 + (labelHash << 1) + uint64(index*2+1)
	mixed ^= mixed >> 30
	mixed *= 0xbf58476d1ce4e5b9
	mixed ^= mixed >> 27
	mixed *= 0x94d049bb133111eb
	mixed ^= mixed >> 31
	return mixed
}

func rng(seed uint64) *rand.Rand {
	return rand.New(rand.NewSource(int64(seed)))
}

func randomIntInclusive(r *rand.Rand, minVal, maxVal int) int {
	if maxVal <= minVal {
		return minVal
	}
	return minVal + r.Intn(maxVal-minVal+1)
}

func chooseDistinctIndices(r *rand.Rand, poolSize, count int) []int {
	if count <= 0 || poolSize <= 0 {
		return nil
	}
	if count > poolSize {
		count = poolSize
	}
	indices := r.Perm(poolSize)
	selected := append([]int(nil), indices[:count]...)
	return selected
}
