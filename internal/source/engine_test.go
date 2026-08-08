package source

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/dharuncs/novel/internal/scraper"
)

func TestSuccessfulCallCleansWatchdog(t *testing.T) {
	plugin, err := LoadScript(`const source={id:"test",name:"Test",version:"1",baseURL:"https://example.com",language:"en",rateLimit:60,needsJS:false,search:function(){return []},novelInfo:function(){return {}},chapterList:function(){return []},chapterContent:function(){return ""}}`, scraper.NewClient())
	if err != nil {
		t.Fatal(err)
	}
	baseline := runtime.NumGoroutine()
	if _, err := plugin.Search(context.Background(), "x", 1); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	if current := runtime.NumGoroutine(); current > baseline+1 {
		t.Fatalf("watchdog goroutine leaked: baseline=%d current=%d", baseline, current)
	}
}
