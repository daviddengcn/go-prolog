package prolog

import (
	"strings"
	"sync"
)

func appendIndent(s, indent string) string {
	return indent + strings.Replace(s, "\n", "\n"+indent, -1)
}

/* name pool: *namePool */

type namePool struct {
	sync.RWMutex
	nameIndex map[string]int
	indexName []string
}

func newNamePool() *namePool {
	return &namePool{nameIndex: make(map[string]int)}
}

func (p *namePool) nameOfIndex(index int) string {
	p.RLock()
	defer p.RUnlock()

	return p.indexName[index]
}

func (p *namePool) indexOfName(name string) int {
	// First try fetch within read-lock
	index, ok := func() (index int, ok bool) {
		p.RLock()
		defer p.RUnlock()

		index, ok = p.nameIndex[name]
		return
	}()

	if ok {
		// if found, return it
		return index
	}

	// otherwise try fetch/create within write-lock
	p.Lock()
	defer p.Unlock()

	// try fetch again, in case it is inserted before atomPool.Lock
	index, ok = p.nameIndex[name]
	if ok {
		return index
	}

	index = len(p.indexName)
	p.indexName = append(p.indexName, name)
	p.nameIndex[name] = index

	return index
}
