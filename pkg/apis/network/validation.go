package network

import (
	"fmt"
	"net"
	"regexp"

	"lcp.io/lcp/lib/api/validation"
)

var networkNameRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$`)

// ValidateNetworkCreate 校验创建网络的参数。
func ValidateNetworkCreate(name string, spec *NetworkSpec) validation.ErrorList {
	var errs validation.ErrorList

	if name == "" {
		errs = append(errs, validation.FieldError{Field: "metadata.name", Message: "is required"})
	} else if !networkNameRegexp.MatchString(name) {
		errs = append(errs, validation.FieldError{Field: "metadata.name", Message: "must be 3-50 lowercase alphanumeric characters or hyphens, starting and ending with alphanumeric"})
	}

	if spec.CIDR != "" {
		if _, _, err := net.ParseCIDR(spec.CIDR); err != nil {
			errs = append(errs, validation.FieldError{Field: "spec.cidr", Message: fmt.Sprintf("invalid CIDR format: %v", err)})
		}
	}

	if spec.MaxSubnets != 0 && (spec.MaxSubnets < 1 || spec.MaxSubnets > 50) {
		errs = append(errs, validation.FieldError{Field: "spec.maxSubnets", Message: "must be between 1 and 50"})
	}

	if len(spec.Description) > 1024 {
		errs = append(errs, validation.FieldError{Field: "spec.description", Message: "must be at most 1024 characters"})
	}

	if spec.Status != "" && spec.Status != "active" && spec.Status != "inactive" {
		errs = append(errs, validation.FieldError{Field: "spec.status", Message: "must be 'active' or 'inactive'"})
	}

	return errs
}

// ValidateNetworkUpdate 校验更新网络的参数。
func ValidateNetworkUpdate(spec *NetworkSpec) validation.ErrorList {
	var errs validation.ErrorList

	if len(spec.Description) > 1024 {
		errs = append(errs, validation.FieldError{Field: "spec.description", Message: "must be at most 1024 characters"})
	}

	if spec.Status != "" && spec.Status != "active" && spec.Status != "inactive" {
		errs = append(errs, validation.FieldError{Field: "spec.status", Message: "must be 'active' or 'inactive'"})
	}

	return errs
}

// ValidateSubnetCreate 校验创建子网的参数。networkCIDR 为所属网络的 CIDR（可选），用于校验子网 CIDR 是否在网络范围内。
func ValidateSubnetCreate(name string, spec *SubnetSpec, existingCIDRs []string, networkCIDR string) validation.ErrorList {
	var errs validation.ErrorList

	if name == "" {
		errs = append(errs, validation.FieldError{Field: "metadata.name", Message: "is required"})
	} else if !networkNameRegexp.MatchString(name) {
		errs = append(errs, validation.FieldError{Field: "metadata.name", Message: "must be 3-50 lowercase alphanumeric characters or hyphens, starting and ending with alphanumeric"})
	}

	if spec.CIDR == "" {
		errs = append(errs, validation.FieldError{Field: "spec.cidr", Message: "is required"})
		return errs
	}

	_, cidrNet, err := net.ParseCIDR(spec.CIDR)
	if err != nil {
		errs = append(errs, validation.FieldError{Field: "spec.cidr", Message: fmt.Sprintf("invalid CIDR format: %v", err)})
		return errs
	}

	// 校验子网 CIDR 在网络 CIDR 范围内
	if networkCIDR != "" {
		_, networkNet, err := net.ParseCIDR(networkCIDR)
		if err == nil {
			ones1, _ := networkNet.Mask.Size()
			ones2, _ := cidrNet.Mask.Size()
			if !networkNet.Contains(cidrNet.IP) || ones2 < ones1 {
				errs = append(errs, validation.FieldError{Field: "spec.cidr", Message: fmt.Sprintf("subnet CIDR %s is not within network CIDR %s", spec.CIDR, networkCIDR)})
			}
		}
	}

	// 校验 gateway 在 CIDR 范围内
	if spec.Gateway != "" {
		gw := net.ParseIP(spec.Gateway)
		if gw == nil {
			errs = append(errs, validation.FieldError{Field: "spec.gateway", Message: "invalid IP address format"})
		} else if !cidrNet.Contains(gw) {
			errs = append(errs, validation.FieldError{Field: "spec.gateway", Message: fmt.Sprintf("gateway %s is not within CIDR %s", spec.Gateway, spec.CIDR)})
		}
	}

	// CIDR 重叠检测
	for _, existing := range existingCIDRs {
		_, existingNet, err := net.ParseCIDR(existing)
		if err != nil {
			continue
		}
		if CIDRsOverlap(cidrNet, existingNet) {
			errs = append(errs, validation.FieldError{Field: "spec.cidr", Message: fmt.Sprintf("CIDR %s overlaps with existing subnet %s", spec.CIDR, existing)})
			break
		}
	}

	if len(spec.Description) > 1024 {
		errs = append(errs, validation.FieldError{Field: "spec.description", Message: "must be at most 1024 characters"})
	}

	return errs
}

// ValidateSubnetUpdate 校验更新子网的参数。
func ValidateSubnetUpdate(spec *SubnetSpec) validation.ErrorList {
	var errs validation.ErrorList

	if len(spec.Description) > 1024 {
		errs = append(errs, validation.FieldError{Field: "spec.description", Message: "must be at most 1024 characters"})
	}

	return errs
}

// ValidateIPAllocationCreate 校验创建 IP 分配的参数。
func ValidateIPAllocationCreate(spec *IPAllocationSpec) validation.ErrorList {
	var errs validation.ErrorList

	if spec.IP == "" {
		errs = append(errs, validation.FieldError{Field: "spec.ip", Message: "is required"})
	} else if ip := net.ParseIP(spec.IP); ip == nil {
		errs = append(errs, validation.FieldError{Field: "spec.ip", Message: "invalid IP address format"})
	}

	if len(spec.Description) > 512 {
		errs = append(errs, validation.FieldError{Field: "spec.description", Message: "must be at most 512 characters"})
	}

	return errs
}

// CIDRsOverlap 检查两个 CIDR 是否存在重叠。
func CIDRsOverlap(a, b *net.IPNet) bool {
	return a.Contains(b.IP) || b.Contains(a.IP)
}
