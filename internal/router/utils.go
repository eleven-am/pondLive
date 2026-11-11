package router

func matchesPrefix(currentPath, targetPath string) bool {
	if targetPath == "/" {
		return currentPath == "/"
	}
	if len(currentPath) < len(targetPath) || currentPath[:len(targetPath)] != targetPath {
		return false
	}
	return len(currentPath) == len(targetPath) || currentPath[len(targetPath)] == '/'
}
