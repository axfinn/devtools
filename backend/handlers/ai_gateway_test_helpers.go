package handlers

import (
	"devtools/utils"
)

// testEncEncryptionService 是测试专用的 AES-256-GCM 加密服务,用固定密钥(非 secret)。
// 各 *Test 文件构造 AIGatewayHandler 时传入,避免在测试里散落 key 字符串字面量。
var testEncEncryptionService = func() *utils.EncryptionService {
	svc, err := utils.NewEncryptionService("devtools-test-master-key")
	if err != nil {
		panic("test encryption service init: " + err.Error())
	}
	return svc
}()
