package control

type MakeDirectoryRequest struct {
	Path string
}

type MakeDirectoryResponse struct {
	CreatedPath string
}

type CheckDeviceRequest struct {
	Path string
}

type CheckDeviceResponse struct {
	Exists bool
}
