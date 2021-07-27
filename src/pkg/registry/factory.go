package registry

import "fmt"

type RegistryStore struct {
	R map[string]*Registry
}

func NewRegistryStore() *RegistryStore {
	var rs RegistryStore

	rs.R = make(map[string]*Registry)
	return &rs
}

func (rs *RegistryStore) Add(name string, r *Registry) {
	rs.R[name] = r
}

func (rs *RegistryStore) Del(name string, r *Registry) {
	rs.R[name] = r
}

func (rs *RegistryStore) Get(name string)(*Registry,error) {
	if v,ok := rs.R[name];ok{
		return v,nil
	}
	return nil,fmt.Errorf("no registry")
}