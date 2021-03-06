package driver

import (
	"errors"
	"log"
	"net"
	"time"

	"fmt"

	"github.com/bwmarrin/snowflake"
	"github.com/docker/go-plugins-helpers/ipam"
)

//RequestAddressType:com.docker.network.gateway

const (
	globalSpace        = "alpha_global"
	localSpace         = "beta_local"
	gatewayRequest     = "com.docker.network.gateway"
	requestAddressType = "RequestAddressType"
)

type IPAM struct {
	verbose     bool
	globalPools []pool
	localPools  []pool
}

func (i *IPAM) GetCapabilities() (*ipam.CapabilitiesResponse, error) {
	if i.verbose {
		log.Println("get capabilities request received")
	}

	return &ipam.CapabilitiesResponse{
		RequiresMACAddress: false,
	}, nil
}

func (i *IPAM) GetDefaultAddressSpaces() (*ipam.AddressSpacesResponse, error) {
	if i.verbose {
		log.Println("get default address spaces request received")
	}

	return &ipam.AddressSpacesResponse{
		GlobalDefaultAddressSpace: globalSpace,
		LocalDefaultAddressSpace:  localSpace,
	}, nil
}

func (i *IPAM) getFreePoolBySpace(space string) (*pool, error) {
	if space == globalSpace {
		for idx, p := range i.globalPools {
			if !p.taken {
				i.globalPools[idx].taken = true

				//if err := i.globalPools[idx].bridgeUp(i.verbose); err != nil {
				//	return nil, err
				//}

				return &i.globalPools[idx], nil
			}
		}
	}

	for idx, p := range i.localPools {
		if !p.taken {
			i.localPools[idx].taken = true

			//if err := i.localPools[idx].bridgeUp(i.verbose); err != nil {
			//	return nil, err
			//}

			return &i.localPools[idx], nil
		}
	}

	return nil, errors.New("No free pools")
}

func (i *IPAM) RequestPool(rq *ipam.RequestPoolRequest) (*ipam.RequestPoolResponse, error) {
	if i.verbose {
		log.Println("pool request received")
		log.Printf("%+v\n", rq)
	}

	data := map[string]string{
		"DNS": "8.8.8.8",
		//"com.docker.network.gateway": "0.0.0.0/0",
	}

	if rq.AddressSpace != "" && rq.Pool != "" {
		if rq.AddressSpace == globalSpace {
			for idx, p := range i.globalPools {
				if p.value == rq.Pool {
					if i.verbose {
						log.Println("global pool found")
					}
					i.globalPools[idx].taken = true

					//if err := i.globalPools[idx].bridgeUp(i.verbose); err != nil {
					//	return nil, err
					//}

					return &ipam.RequestPoolResponse{
						Pool:   rq.Pool,
						PoolID: p.pid,
						Data:   data,
					}, nil
				}
			}
		}
		if rq.AddressSpace == localSpace {
			for idx, p := range i.localPools {
				if p.value == rq.Pool {
					if i.verbose {
						log.Println("local pool found")
					}

					i.localPools[idx].taken = true

					//if err := i.localPools[idx].bridgeUp(i.verbose); err != nil {
					//	return nil, err
					//}

					return &ipam.RequestPoolResponse{
						Pool:   rq.Pool,
						PoolID: p.pid,
						Data:   data,
					}, nil
				}
			}
		}
	}

	p, err := i.getFreePoolBySpace(rq.AddressSpace)
	if err != nil {
		return nil, err
	}

	return &ipam.RequestPoolResponse{
		Data:   data,
		Pool:   p.value,
		PoolID: p.pid,
	}, nil
}

func (i *IPAM) ReleasePool(rq *ipam.ReleasePoolRequest) error {
	if i.verbose {
		log.Println("release pool request received")
		log.Printf("%+v\n", rq)
	}

	for idx, p := range i.globalPools {
		if p.pid == rq.PoolID {
			i.globalPools[idx].taken = false
			//(*i.globalPools[idx].link).DeleteLink()
			//(*i.globalPools[idx].bridge).DeleteLink()
			return nil
		}
	}

	for idx, p := range i.localPools {
		if p.pid == rq.PoolID {
			i.localPools[idx].taken = false
			//(*i.localPools[idx].link).DeleteLink()
			//(*i.localPools[idx].bridge).DeleteLink()
			return nil
		}
	}

	return nil
}

func (i *IPAM) getGatewayByPoolID(pid string) (string, error) {
	for _, p := range i.globalPools {
		if p.pid == pid {
			return p.gateway, nil
		}
	}

	for _, p := range i.localPools {
		if p.pid == pid {
			return p.gateway, nil
		}
	}

	return "", fmt.Errorf("Gateway for pool %s not found\n", pid)
}

func (i *IPAM) RequestAddress(rq *ipam.RequestAddressRequest) (*ipam.RequestAddressResponse, error) {
	if i.verbose {
		log.Println("address request received")
		log.Printf("%+v\n", rq)
	}

	if t, ok := rq.Options[requestAddressType]; ok {
		switch t {
		case gatewayRequest:
			addr, err := i.getGatewayByPoolID(rq.PoolID)
			if err != nil {
				return nil, err
			}
			return &ipam.RequestAddressResponse{
				Address: addr,
			}, nil
			//return &ipam.RequestAddressResponse{
			//	Address: "172.38.0.0/16",
			//}, nil
		}
	}

	if rq.Address != "" {
		return &ipam.RequestAddressResponse{
			Address: rq.Address,
			//Data:    map[string]string{},
		}, nil
	}

	ip, err := i.getFreeIPByPoolID(rq.PoolID)
	if err != nil {
		return nil, err
	}

	return &ipam.RequestAddressResponse{
		//Data:    map[string]string{},
		Address: ip + "/24",
	}, nil
}

func (i *IPAM) getFreeIPByPoolID(poolID string) (string, error) {
	for idx, p := range i.globalPools {
		if p.pid == poolID {
			if len(p.ips) > 0 {
				var x net.IP
				x, i.globalPools[idx].ips = p.ips[len(p.ips)-1], p.ips[:len(p.ips)-1]
				return x.String(), nil
			}
		}
	}

	for idx, p := range i.localPools {
		if p.pid == poolID {
			if len(p.ips) > 0 {
				var x net.IP
				x, i.localPools[idx].ips = p.ips[len(p.ips)-1], p.ips[:len(p.ips)-1]
				return x.String(), nil
			}
		}
	}

	return "", errors.New("No ip address available")
}

func (i *IPAM) ReleaseAddress(rq *ipam.ReleaseAddressRequest) error {
	if i.verbose {
		log.Println("release address request received")
		log.Printf("%+v\n", rq)
	}

	ip := net.ParseIP(rq.Address)

	for idx, p := range i.globalPools {
		if p.pid == rq.PoolID {
			i.globalPools[idx].ips = append(p.ips, ip)
			return nil
		}
	}

	for idx, p := range i.localPools {
		if p.pid == rq.PoolID {
			i.localPools[idx].ips = append(p.ips, ip)
			return nil
		}
	}

	return errors.New("No pool found")
}

func MakeIPAM(verbose bool, globalPools, localPools []string) (*IPAM, error) {
	i := IPAM{
		verbose: verbose,
	}

	node, _ := snowflake.NewNode(time.Now().UnixNano() % 1022)
	var localIPs, globalIPs int

	for _, globalPool := range globalPools {
		p, err := makePool(verbose, globalPool, node.Generate().Base36())
		if err != nil {
			return &i, err
		}
		globalIPs += len(p.ips)
		i.globalPools = append(i.globalPools, *p)
	}

	for _, localPool := range localPools {
		p, err := makePool(verbose, localPool, node.Generate().Base36())
		if err != nil {
			return &i, err
		}
		localIPs += len(p.ips)
		i.localPools = append(i.localPools, *p)
	}

	if verbose {
		fmt.Printf("Local pools: %d, Local IPs: %d\nGlobal pools: %d, Global IPs: %d\n",
			len(i.localPools), localIPs, len(i.globalPools), globalIPs)
	}

	return &i, nil
}
