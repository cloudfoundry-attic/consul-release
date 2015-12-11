package bosh

import (
	"sort"
	"strconv"
)

type Stemcell struct {
	Name     string
	Versions []string
}

func NewStemcell() Stemcell {
	return Stemcell{}
}

func (s Stemcell) Latest() string {
	tmp := []int{}

	for _, version := range s.Versions {
		num, err := strconv.Atoi(version)
		if err != nil {
			return s.Versions[len(s.Versions)-1]
		}
		tmp = append(tmp, num)
	}
	sort.Ints(tmp)

	s.Versions = []string{}

	for _, version := range tmp {
		s.Versions = append(s.Versions, strconv.Itoa(version))
	}

	return s.Versions[len(s.Versions)-1]
}
