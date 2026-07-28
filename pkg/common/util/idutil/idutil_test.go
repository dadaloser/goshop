package idutil

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetUUID(t *testing.T) {
	//fmt.Println(GetUUID(""))
}

func TestGetUUID36(t *testing.T) {
	fmt.Println(GetUUID36(""))
}

func TestGetManyUuid(t *testing.T) {
	for i := 0; i < 10000; i++ {
		//testID := GetUUID("")
		//if len(testID) != 12 {
		//	t.Errorf("GetUUID failed, expected uuid length 12, got: %d", len(testID))
		//}
	}
}

func TestRandString(t *testing.T) {
	str := randString(Alphabet62, 50)
	assert.Equal(t, 50, len(str))
	t.Log(str)

	str = randString(Alphabet62, 255)
	assert.Equal(t, 255, len(str))
	t.Log(str)

	if got := randString("", 12); got != "" {
		t.Fatalf("randString(empty, 12) = %q, want empty", got)
	}
	if got := randString(Alphabet62, 0); got != "" {
		t.Fatalf("randString(alphabet, 0) = %q, want empty", got)
	}
	if strings.IndexFunc(str, func(r rune) bool { return !strings.ContainsRune(Alphabet62, r) }) >= 0 {
		t.Fatalf("randString() produced character outside alphabet: %q", str)
	}
}
