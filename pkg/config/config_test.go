package config

import "testing"

func TestGetConfig(t *testing.T) {
	etcds := []Etcd{
		{
			Ip:   "127.0.0.1",
			Port: "2379",
		},
	}
	for i, v := range GetEtcds() {
		if etcds[i].Ip != v.Ip || etcds[i].Port != v.Port {
			t.Errorf("error")
		}
	}
}
