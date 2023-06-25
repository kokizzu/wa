// 版权 @2023 凹语言 作者。保留所有权利。

package appplay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"wa-lang.org/wa/api"
)

func (p *WebServer) fmtHandler(w http.ResponseWriter, r *http.Request) {
	var (
		in  = []byte(r.FormValue("body"))
		err error
	)

	resp, err := p.fmtCode(in)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("error encoding response: %v", err), http.StatusInternalServerError)
		return
	}
}

func (p *WebServer) fmtCode(code []byte) (*fmtResponse, error) {
	filename := "prog.wa"
	if bytes.Contains(code, []byte("【启】：")) {
		filename = "prog.wz"
	}

	output, err := api.FormatCode(filename, string(code))
	if err != nil {
		resp := &fmtResponse{
			Error: err.Error(),
		}
		return resp, nil
	}

	resp := &fmtResponse{
		Body: string(output),
	}

	return resp, nil
}
