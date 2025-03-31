package webservice

import (
	"os"
	"os/user"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGatewayAgentString(t *testing.T) {
	var str = userAgent()

	var usr, err = user.Current()
	assert.Nil(t, err)
	if usr.Name != "" {
		assert.Equal(t, httpAgentPrefix+"/"+usr.Name, str)
	} else {
		assert.Equal(t, httpAgentPrefix+"/"+usr.Username, str)
	}

	os.Setenv("SYSTEM", "system")
	os.Setenv("COMPONENT", "component")

	str = userAgent()
	assert.Equal(t, httpAgentPrefix+"/system/component", str)
}

func TestCombineURL(t *testing.T) {
	var p string

	p = combineURL("https://localhost", "")
	assert.Equal(t, "https://localhost/", p)
	p = combineURL("https://localhost/", "")
	assert.Equal(t, "https://localhost/", p)
	p = combineURL("https://localhost/", "/")
	assert.Equal(t, "https://localhost/", p)
	p = combineURL("https://localhosta", "b")
	assert.Equal(t, "https://localhosta/b", p)
	p = combineURL("https://localhosta/#coiso", "b")
	assert.Equal(t, "https://localhosta/#coiso/b", p)
	p = combineURL("http://localhost/a", "b")
	assert.Equal(t, "http://localhost/a/b", p)
	p = combineURL("http://localhost", "a/b")
	assert.Equal(t, "http://localhost/a/b", p)
	p = combineURL("http://localhost/", "a/c/")
	assert.Equal(t, "http://localhost/a/c/", p)
}

func BenchmarkCombineURL(b *testing.B) {
	var p string

	b.Run("simple", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			p = combineURL("https://localhost", "/asd/bas/"+strconv.Itoa(i))
			if p == "https://localhost/asd/bas/102947102947102938012983" {
				b.FailNow()
			}
		}
	})
}
