package source

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/dharuncs/novel/internal/scraper"
	"github.com/dop251/goja"
)

const (
	idleTimeout  = 30 * time.Second
	outerTimeout = 5 * time.Minute
)

type Plugin struct {
	Metadata Metadata
	script   string
	client   *scraper.Client
	OnProgress func(url string)
}
type execution struct {
	source  goja.Value
	runtime *goja.Runtime
	cancel  context.CancelFunc
	idle    *time.Timer
	done    chan struct{}
}

func (run *execution) close() {
	run.cancel()
	if !run.idle.Stop() {
		select {
		case <-run.idle.C:
		default:
		}
	}
	close(run.done)
}

func LoadScript(script string, client *scraper.Client) (*Plugin, error) {
	plugin := &Plugin{script: script, client: client}
	run, err := plugin.run(context.Background())
	if err != nil {
		return nil, fmt.Errorf("load plugin: %w", err)
	}
	defer run.close()
	metadata := run.source.ToObject(run.runtime)
	plugin.Metadata = Metadata{ID: metadata.Get("id").String(), Name: metadata.Get("name").String(), Version: metadata.Get("version").String(), BaseURL: metadata.Get("baseURL").String(), Language: metadata.Get("language").String(), NeedsJS: metadata.Get("needsJS").ToBoolean(), RateLimit: int(metadata.Get("rateLimit").ToInteger())}
	if err := plugin.validate(); err != nil {
		return nil, err
	}
	return plugin, nil
}

func (plugin *Plugin) Search(ctx context.Context, query string, page int) ([]SearchResult, error) {
	var results []SearchResult
	err := plugin.call(ctx, "search", &results, query, page)
	return results, err
}
func (plugin *Plugin) NovelInfo(ctx context.Context, novelURL string) (Novel, error) {
	var novel Novel
	err := plugin.call(ctx, "novelInfo", &novel, novelURL)
	return novel, err
}
func (plugin *Plugin) ChapterList(ctx context.Context, novelURL string) ([]Chapter, error) {
	var chapters []Chapter
	err := plugin.call(ctx, "chapterList", &chapters, novelURL)
	return chapters, err
}
func (plugin *Plugin) ChapterContent(ctx context.Context, chapterURL string) (string, error) {
	var content string
	err := plugin.call(ctx, "chapterContent", &content, chapterURL)
	return content, err
}

func (plugin *Plugin) call(ctx context.Context, method string, output any, arguments ...any) error {
	run, err := plugin.run(ctx)
	if err != nil {
		return err
	}
	defer run.close()
	source, runtime := run.source, run.runtime
	object := source.ToObject(nil)
	function, ok := goja.AssertFunction(object.Get(method))
	if !ok {
		return fmt.Errorf("plugin %q lacks %s()", plugin.Metadata.ID, method)
	}
	values := make([]goja.Value, len(arguments))
	for index, argument := range arguments {
		values[index] = runtime.ToValue(argument)
	}
	result, err := function(source, values...)
	if err != nil {
		return fmt.Errorf("%s: %w", method, err)
	}
	exported := result.Export()
	if exported == nil {
		return nil
	}
	if strPtr, ok := output.(*string); ok {
		if str, ok := exported.(string); ok {
			*strPtr = str
			return nil
		}
	}
	data, err := json.Marshal(exported)
	if err != nil {
		return fmt.Errorf("marshal %s result: %w", method, err)
	}
	if err := json.Unmarshal(data, output); err != nil {
		return fmt.Errorf("decode %s result: %w", method, err)
	}
	return nil
}

func (plugin *Plugin) run(parent context.Context) (*execution, error) {
	ctx, cancel := context.WithTimeout(parent, outerTimeout)
	idle := time.NewTimer(idleTimeout)
	runtime := goja.New()
	runtime.Set("fetch", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(runtime.NewTypeError("fetch requires a URL"))
		}
		target := call.Argument(0).String()
		if err := plugin.allowURL(target); err != nil {
			panic(runtime.NewTypeError(err.Error()))
		}
		if plugin.OnProgress != nil {
			plugin.OnProgress(target)
		}
		html, err := plugin.client.Fetch(ctx, plugin.Metadata.ID, plugin.Metadata.RateLimit, plugin.Metadata.BaseURL, target)
		if err != nil {
			panic(runtime.NewGoError(err))
		}
		if !idle.Stop() {
			select {
			case <-idle.C:
			default:
			}
		}
		idle.Reset(idleTimeout)
		return runtime.ToValue(html)
	})
	runtime.Set("selectHTML", func(html, selector string) ([]SelectorMatch, error) {
		selections, err := scraper.SelectHTML(html, selector)
		if err != nil {
			return nil, err
		}
		matches := make([]SelectorMatch, len(selections))
		for index, selection := range selections {
			matches[index] = SelectorMatch{Text: strings.TrimSpace(selection.Text), HTML: selection.HTML, Attrs: selection.Attrs}
		}
		return matches, nil
	})
	runtime.Set("log", func(args ...any) { fmt.Println(args...) })
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			runtime.Interrupt(fmt.Errorf("plugin exceeded %s execution budget: %w", outerTimeout, ctx.Err()))
		case <-idle.C:
			runtime.Interrupt(fmt.Errorf("plugin idle: no fetch() activity for %s", idleTimeout))
		case <-done:
		}
	}()
	_, err := runtime.RunString(plugin.script)
	if err != nil {
		cancel()
		idle.Stop()
		close(done)
		return nil, err
	}
	value := runtime.Get("source")
	if goja.IsUndefined(value) || goja.IsNull(value) {
		cancel()
		idle.Stop()
		close(done)
		return nil, fmt.Errorf("plugin must define global source object")
	}
	return &execution{source: value, runtime: runtime, cancel: cancel, idle: idle, done: done}, nil
}

func (plugin *Plugin) validate() error {
	if plugin.Metadata.ID == "" || plugin.Metadata.Name == "" || plugin.Metadata.BaseURL == "" {
		return fmt.Errorf("plugin metadata requires id, name, and baseURL")
	}
	base, err := url.Parse(plugin.Metadata.BaseURL)
	if err != nil || base.Scheme != "https" || base.Host == "" {
		return fmt.Errorf("plugin %q has invalid baseURL", plugin.Metadata.ID)
	}
	for _, method := range []string{"search", "novelInfo", "chapterList", "chapterContent"} {
		run, err := plugin.run(context.Background())
		if err != nil {
			return err
		}
		if _, ok := goja.AssertFunction(run.source.ToObject(nil).Get(method)); !ok {
			run.close()
			return fmt.Errorf("plugin %q lacks %s()", plugin.Metadata.ID, method)
		}
		run.close()
	}
	return nil
}
func (plugin *Plugin) allowURL(target string) error {
	parsed, err := url.Parse(target)
	if err != nil {
		return fmt.Errorf("invalid fetch URL")
	}
	base, _ := url.Parse(plugin.Metadata.BaseURL)
	if parsed.Scheme != "https" || !strings.EqualFold(parsed.Host, base.Host) {
		return fmt.Errorf("URL is outside plugin baseURL")
	}
	return nil
}
