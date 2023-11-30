package seal

type Config struct {
	Content  map[string]ContentFunc // key is file extension
	Handlers map[string]HandlerGen  // key is file extension or full filename
}
