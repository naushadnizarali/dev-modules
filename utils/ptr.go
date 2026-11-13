package utils

import "strconv"

func StringToUint(s string) uint {
	val, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return uint(val)
}

func StringToUintPtr(s string) *uint {
	num, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return nil
	}
	n := uint(num)
	return &n
}

func StringToUint32(s string) uint32 {
	num, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0
	}
	return uint32(num)
}

func StringToUint32Ptr(s string) *uint32 {
	num, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return nil
	}
	n := uint32(num)
	return &n
}

func StringToUint64(s string) uint64 {
	num, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return num
}

func StringPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func UIntPtr(i *uint) uint {
	return *i
}
