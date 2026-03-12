package utils

// ContainsIgnoreCase 检查字符串是否包含另一个字符串（忽略大小写）
func ContainsIgnoreCase(s, substr string) bool {
	sLower := ToLowerCase(s)
	substrLower := ToLowerCase(substr)

	if len(substrLower) == 0 {
		return true
	}

	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		if sLower[i:i+len(substrLower)] == substrLower {
			return true
		}
	}
	return false
}

// ToLowerCase 将字符串转换为小写
func ToLowerCase(s string) string {
	var result []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result = append(result, c+'a'-'A')
		} else {
			result = append(result, c)
		}
	}
	return string(result)
}
