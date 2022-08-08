package globalvar

import(
	"github.com/heckintosh/nuclei/v2/pkg/output"
)
var globalRes []*output.ResultEvent

func Set(res []*output.ResultEvent){
	globalRes = res
}
func Get() []*output.ResultEvent{
	return globalRes
}