package utils

import (
	"fmt"
	"math"
	"math/big"
	"net"
)

// generateSubnets 生成指定数量的子网CIDR块
func GenerateSubnets(vpcCIDR string, subnetCount int) ([]string, error) {
	_, vpcNet, err := net.ParseCIDR(vpcCIDR)
	if err != nil {
		return nil, fmt.Errorf("invalid VPC CIDR: %v", err)
	}

	// 计算每个子网的前缀长度
	subnetMaskSize, totalMaskSize := vpcNet.Mask.Size()
	requiredBits := int(math.Ceil(math.Log2(float64(subnetCount))))
	newPrefix := subnetMaskSize + requiredBits

	// 检查子网是否能容纳指定数量的子网
	if newPrefix > totalMaskSize {
		return nil, fmt.Errorf("subnet count too large for the given VPC CIDR. Maximum subnets that can be generated: %d", 1<<(totalMaskSize-subnetMaskSize))
	}

	// 使用go-cidr库生成子网
	subnets := []string{}
	for i := 0; i < subnetCount; i++ {
		subnet, err := subnet(vpcNet, requiredBits, i)
		if err != nil {
			return nil, fmt.Errorf("failed to generate subnet: %v", err)
		}
		subnets = append(subnets, subnet.String())
	}

	return subnets, nil
}

// subnet takes a parent CIDR range and creates a subnet within it
// with the given number of additional prefix bits and the given
// network number.
//
// For example, 10.3.0.0/16, extended by 8 bits, with a network number
// of 5, becomes 10.3.5.0/24 .
func subnet(base *net.IPNet, newBits int, num int) (*net.IPNet, error) {
	return subnetBig(base, newBits, big.NewInt(int64(num)))
}

// subnetBig takes a parent CIDR range and creates a subnet within it with the
// given number of additional prefix bits and the given network number. It
// differs from Subnet in that it takes a *big.Int for the num, instead of an int.
//
// For example, 10.3.0.0/16, extended by 8 bits, with a network number of 5,
// becomes 10.3.5.0/24 .
func subnetBig(base *net.IPNet, newBits int, num *big.Int) (*net.IPNet, error) {
	ip := base.IP
	mask := base.Mask

	parentLen, addrLen := mask.Size()
	newPrefixLen := parentLen + newBits

	if newPrefixLen > addrLen {
		return nil, fmt.Errorf("insufficient address space to extend prefix of %d by %d", parentLen, newBits)
	}

	maxNetNum := uint64(1<<uint64(newBits)) - 1
	if num.Uint64() > maxNetNum {
		return nil, fmt.Errorf("prefix extension of %d does not accommodate a subnet numbered %d", newBits, num)
	}

	return &net.IPNet{
		IP:   insertNumIntoIP(ip, num, newPrefixLen),
		Mask: net.CIDRMask(newPrefixLen, addrLen),
	}, nil
}

func ipToInt(ip net.IP) (*big.Int, int) {
	val := &big.Int{}
	val.SetBytes([]byte(ip))
	if len(ip) == net.IPv4len {
		return val, 32
	} else if len(ip) == net.IPv6len {
		return val, 128
	} else {
		return nil, 0
	}
}

func intToIP(ipInt *big.Int, bits int) net.IP {
	ipBytes := ipInt.Bytes()
	ret := make([]byte, bits/8)
	// Pack our IP bytes into the end of the return array,
	// since big.Int.Bytes() removes front zero padding.
	for i := 1; i <= len(ipBytes); i++ {
		ret[len(ret)-i] = ipBytes[len(ipBytes)-i]
	}
	return net.IP(ret)
}

func insertNumIntoIP(ip net.IP, bigNum *big.Int, prefixLen int) net.IP {
	ipInt, totalBits := ipToInt(ip)
	bigNum.Lsh(bigNum, uint(totalBits-prefixLen))
	ipInt.Or(ipInt, bigNum)
	return intToIP(ipInt, totalBits)
}
