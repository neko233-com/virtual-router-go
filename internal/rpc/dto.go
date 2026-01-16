package rpc

type RpcRequest struct {
	FromRouteId        string   `json:"fromRouteId"`
	ToRouteId          string   `json:"toRouteId"`
	RpcUid             string   `json:"rpcUid"`
	StartTimeMs        int64    `json:"startTimeMs"`
	PacketId           int      `json:"packetId"`
	MethodArgsJsonList []string `json:"methodArgsJsonList"`
}

type RpcResponse struct {
	RpcUid         string `json:"rpcUid"`
	ErrorFlag      bool   `json:"errorFlag"`
	ErrorMsg       string `json:"errorMsg"`
	StartTimeMs    int64  `json:"startTimeMs"`
	PacketId       int    `json:"packetId"`
	ResultValueStr string `json:"resultValueStr"`
}
