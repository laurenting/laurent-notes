# laurent-notes
最近遇到了重复的bug却没有想起来耽误了很长时间

记录一些不经常遇到（容易忘记）的问题方便查找

## GO

### 1.Tue Jun 08 2021 21:32:38 初始化gorm

**context**： *gin-vue-admin*

**description**： 在service写好API函数后随即写了xx_test.go,执行测试函数的时候的报错

**cause**：被调用的函数内有用到全局的 gorm，由于没有初始化所以找不到

**solution**：在测试时做好初始化，后来发现既然是在写api就写完后用 *swagger-go* 直接调用接口测试

```go
GOROOT=/usr/local/go #gosetup
GOPATH=/Users/laurent/go #gosetup
/usr/local/go/bin/go test -c -o /private/var/folders/xs/rz1dk04s1m1gv98fyc9h9b0m0000gn/T/___TestSyncJDChatGroup_in_code_haxii_com_thor_sif_service code.haxii.com/thor/sif/service #gosetup
/usr/local/go/bin/go tool test2json -t /private/var/folders/xs/rz1dk04s1m1gv98fyc9h9b0m0000gn/T/___TestSyncJDChatGroup_in_code_haxii_com_thor_sif_service -test.v -test.paniconexit0 -test.run ^\QTestSyncJDChatGroup\E$
=== RUN   TestSyncJDChatGroup
--- FAIL: TestSyncJDChatGroup (0.00s)
panic: runtime error: invalid memory address or nil pointer dereference [recovered]
	panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0x1 addr=0x10 pc=0x4728e43]

goroutine 21 [running]:
testing.tRunner.func1.2(0x49b9920, 0x526c7e0)
	/usr/local/go/src/testing/testing.go:1144 +0x332
testing.tRunner.func1(0xc000082f00)
	/usr/local/go/src/testing/testing.go:1147 +0x4b6
panic(0x49b9920, 0x526c7e0)
	/usr/local/go/src/runtime/panic.go:965 +0x1b9
code.haxii.com/thor/cookie.(*Cache).Query(0x0, 0x4acd9a4, 0x7, 0x4ace4a5, 0x8, 0x0, 0x4124c80, 0x503dea0)
	/Users/laurent/go/pkg/mod/code.haxii.com/thor/cookie@v0.1.0/cache.go:56 +0x43
code.haxii.com/thor/cookie.(*Cache).Get(...)
	/Users/laurent/go/pkg/mod/code.haxii.com/thor/cookie@v0.1.0/cache.go:41
code.haxii.com/thor/sif/service.SyncJDChatGroup(0xc000484cf0, 0x1, 0x1, 0x1445ed5421144, 0xc0165331d8)
	/Users/laurent/Downloads/thor/sif/service/jd_chat_group.go:24 +0x12a
code.haxii.com/thor/sif/service.TestSyncJDChatGroup(0xc000082f00)
	/Users/laurent/Downloads/thor/sif/service/jd_chat_goup_test.go:10 +0x65
testing.tRunner(0xc000082f00, 0x4b0c100)
	/usr/local/go/src/testing/testing.go:1194 +0xef
created by testing.(*T).Run
	/usr/local/go/src/testing/testing.go:1239 +0x2b3


Process finished with exit code 1

```

### 2.Tue Jun 08 2021 21:48:11 mysql事物

**context**：*gin-vue-admin*

**description**：第一次接触到 *mysql* 事务(*transaction*)， 项目里用的 gorm ，使用手动流程，以防万一没有结束事物就使用 defer 来做最终的操作

```go
package service

import (
	"fmt"
	"io"
	"testing"
)

func a() error {
	var err error
	defer func() {
		if err != nil {
			fmt.Println("defer1", err)
		} else {
			fmt.Println("ok")
		}
	}()

	err = io.ErrNoProgress

	defer func() {
		err = io.ErrShortWrite
		if err != nil {
			fmt.Println("defer2", err)
		} else {
			fmt.Println("ok")
		}
	}()
	err = io.ErrClosedPipe

	fmt.Println("err", err)
	return err
}
func TestDefer(t *testing.T) {
	fmt.Println(a())
}

```

### 3.Tue Jun 08 2021 22:00:38 join查询

**context:** gorm

**description:** join查询 默认为left join

**solution:** 按照需要更改即可

```go
db.Table("go_service_info").Select("go_service_info.serviceId as service_id, go_service_info.serviceName as service_name, go_system_info.systemId as system_id, go_system_info.systemName as system_name").Joins("left join go_system_info on go_service_info.systemId = go_system_info.systemId where go_service_info.serviceId <> ? and go_system_info.systemId = ?", "xxx", "xxx").Scan(&results)

```

### 4.Wed Jun 09 2021 14:38:03 读写🔒

**目的**：某个操作被多个执行人调用，如果同时读取数据是没有问题的，但如果同时写数据就会有冲突

**方法**一：手动控制每次开锁关锁

```go
type handler struct {
	logger       log.Logger
	who          string
	jdAPI        *JDAPI
	uploadClient *CookieUploadClient

	uploadRateLimitMu  sync.RWMutex    // 读写锁
	uploadRateLimitMap map[string]uploadRateLimit
}
//……//
func (h *handler) rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie := r.Header.Get("Cookie")
		if len(cookie) > 0 {
			h.uploadRateLimitMu.RLock()
			rateLimit := h.uploadRateLimitMap[cookie]
			h.uploadRateLimitMu.RUnlock()
			if time.Now().Sub(rateLimit.LastUploadTime) < cookieUploadRateLimitDuration {
				if _, err := fmt.Fprintf(w, autoRefreshHTML, "same cookie upload rate limit"); err != nil {
					h.logger.Error(h.who, err, "fail to respond %s", r.RemoteAddr)
				}
				return
			}
		}
		next.ServeHTTP(w, r)
		if len(cookie) > 0 {
			h.uploadRateLimitMu.Lock()     // 更改前把数据锁住，只能这这次操作更改
			h.uploadRateLimitMap[cookie] = uploadRateLimit{LastUploadTime: time.Now()}
			h.uploadRateLimitMu.Unlock()	// 更改后把数据解锁
		}
	})
}
//……//
```

**方法二**：*sync.Map* go的sync库封装了一个Map

把它当作一个更高效的redis使用

```
// Map is like a Go map[interface{}]interface{} but is safe for concurrent use
// by multiple goroutines without additional locking or coordination.
// Loads, stores, and deletes run in amortized constant time.
```

```go
type shopInfo struct {
   logger       log.Logger
   cookieClient *client.JDCookieClient

   cachedMallID sync.Map
}
//……//
func (s *shopInfo) getMallID(cookie *client.JDCookie) (string, error) {
	_mallID, exists := s.cachedMallID.Load(cookie.ShopID)
	if exists {
		return _mallID.(string), nil
	}
	mallID, err := s.queryMallID(cookie)
	if err != nil {
		return "", err
	}
	_mallID, _ = s.cachedMallID.LoadOrStore(cookie.ShopID, mallID)
	return _mallID.(string), nil
}
//……//
```

```go
Methods:
  Load(key interface{}) (value interface{}, ok bool)
  Store(key interface{}, value interface{})
  LoadOrStore(key interface{}, value interface{}) (actual interface{}, loaded bool)
  LoadAndDelete(key interface{}) (value interface{}, loaded bool)
  Delete(key interface{})
  Range(f func(key interface{}, value interface{}) bool)
  missLocked()
  dirtyLocked()
```



## JS

### 1.Tue Jun 08 2021 22:00:38 el-select

描述：el-select 如果带有 filterable=true, 当选择之后，切换窗口然后回到原来的页面会自动聚焦到select，下拉列表自动打开 该行为不应该出现 因为已经选择过了 只是没有blur

解决: https://segmentfault.com/q/1010000024421815?utm_source=tag-newest

选中后延时取消聚焦

```javascript
setTimeout(() => {
    this.$refs.el-select.blur()
}, 50)
```

写一个vue指令 控制select行为

```js
Vue.directive('select-blur', {
  bind(el, binding) {
    let inputDom = el.getElementsByTagName('input')[0]
    if (!inputDom) return 
    let isOpen = false
    inputDom.addEventListener('focus', function (event) {
      if (isOpen) {
        inputDom.blur()
      } else {
        isOpen = true
      }
    })
  }
})
```

## Dashboard

### 1.Wed Jun 09 2021 09:40:15 resp 403

**context**:gin-vue-admin

**description**:测试时 在本地启动后端 http://127.0.0.1:8888  前端调用api 正常使用；把后端部署到正式服务器上后 返回 403 

**cause**:需要在sif的前端设置上 新路由路径 /* 以及权限配置

