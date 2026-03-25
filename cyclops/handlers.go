package cyclops

import "errors"
import "strings"
import "io"
import "fmt"
import "net/http"
import "encoding/json"
import "github.com/go-chi/chi/v5"
import "github.com/indexdata/ccms"

type TagList struct {
	Tags []any `json:"tags"`
	// No other elements yet, but use a structure for future expansion
}

func (server *ModCyclopsServer) handleShowTags(w http.ResponseWriter, req *http.Request, caption string) error {
	resp, err := server.sendToCCMS(caption, "show tags;")
	if err != nil {
		return err
	}

	result := readResults(resp)[0]
	tags := make([]any, 0)
	for val := range result.Data() {
		tags = append(tags, val.Values()[0])
	}
	tagList := TagList{Tags: tags}
	return respondWithJSON(w, tagList, caption)
}

// -----------------------------------------------------------------------------

type DefineTag struct {
	Name string `json:"name"`
}

func (server *ModCyclopsServer) handleDefineTag(w http.ResponseWriter, req *http.Request, caption string) error {
	var tag DefineTag
	err := unmarshalBody(req, &tag)
	if err != nil {
		return fmt.Errorf("%s: %w", caption, err)
	}

	command := "define tag " + tag.Name
	server.Log("command", command)

	resp, err := server.sendToCCMS(caption+" "+tag.Name, command)
	if err != nil {
		return err
	}
	fmt.Printf("%s response: %+v\n", caption, resp)

	w.WriteHeader(http.StatusNoContent)
	return nil
}

// -----------------------------------------------------------------------------

type FilterList struct {
	Filters []any `json:"filters"`
	// No other elements yet, but use a structure for future expansion
}

func (server *ModCyclopsServer) handleShowFilters(w http.ResponseWriter, req *http.Request, caption string) error {
	resp, err := server.sendToCCMS(caption, "show filters;")
	if err != nil {
		return err
	}

	result := readResults(resp)[0]
	filters := make([]any, 0)
	for val := range result.Data() {
		filters = append(filters, val.Values()[0])
	}
	filterList := FilterList{Filters: filters}
	return respondWithJSON(w, filterList, caption)
}

// -----------------------------------------------------------------------------

type DefineFilter struct {
	Name     string `json:"name"`
	Cond     string `json:"cond"`
	Template string `json:"template"`
}

func (server *ModCyclopsServer) handleDefineFilter(w http.ResponseWriter, req *http.Request, caption string) error {
	var filter DefineFilter
	err := unmarshalBody(req, &filter)
	if err != nil {
		return fmt.Errorf("%s: %w", caption, err)
	}

	command := "define filter " + filter.Name
	if filter.Cond != "" {
		command += " where " + filter.Cond
	}
	if filter.Template != "" {
		command += " template " + filter.Template
	}
	server.Log("command", command)

	resp, err := server.sendToCCMS(caption+" "+filter.Name, command)
	if err != nil {
		return err
	}
	fmt.Printf("%s response: %+v\n", caption, resp)

	w.WriteHeader(http.StatusNoContent)
	return nil
}

// -----------------------------------------------------------------------------

type SetList struct {
	Sets []any `json:"sets"`
	// No other elements yet, but use a structure for future expansion
}

func (server *ModCyclopsServer) handleShowSets(w http.ResponseWriter, req *http.Request, caption string) error {
	resp, err := server.sendToCCMS(caption, "show sets;")
	if err != nil {
		return fmt.Errorf("could not fetch show-sets response: %w", err)
	}

	result := readResults(resp)[0]
	sets := make([]any, 0)
	for val := range result.Data() {
		sets = append(sets, val.Values()[0])
	}
	setList := SetList{Sets: sets}
	return respondWithJSON(w, setList, caption)
}

// -----------------------------------------------------------------------------

type CreateSet struct {
	Name string `json:"name"`
}

func (server *ModCyclopsServer) handleCreateSet(w http.ResponseWriter, req *http.Request, caption string) error {
	var set CreateSet
	err := unmarshalBody(req, &set)
	if err != nil {
		return fmt.Errorf("%s: %w", caption, err)
	}

	command := "create set " + set.Name + ";"
	server.Log("command", command)

	resp, err := server.sendToCCMS(caption+" "+set.Name, command)
	if err != nil {
		return err
	}
	fmt.Printf("%s response: %+v\n", caption, resp)

	w.WriteHeader(http.StatusNoContent)
	return nil
}

// -----------------------------------------------------------------------------

func makeSelectClause(fields, setName, cond, filter, tag, omitTag, sort, limit, offset string) (string, error) {
	var b strings.Builder

	if fields == "" {
		return "", errors.New("no 'fields' parameter supplied")
	}
	b.WriteString("select ")
	b.WriteString(fields)

	b.WriteString(" from ")
	b.WriteString(setName)

	if cond != "" {
		b.WriteString(" where ")
		b.WriteString(cond)
	}

	if filter != "" {
		b.WriteString(" filter ")
		b.WriteString(filter)
	}

	if tag != "" && omitTag != "" {
		return "", errors.New("both 'tag' and 'omitTag' parameters supplied")
	}

	if tag != "" {
		b.WriteString(" tag ")
		b.WriteString(tag)
	} else if omitTag != "" {
		b.WriteString(" tag not ")
		b.WriteString(omitTag)
	}

	if sort != "" {
		b.WriteString(" order by ")
		b.WriteString(sort)
	}

	if limit == "" {
		limit = "100"
	}
	b.WriteString(" limit ")
	b.WriteString(limit)

	if offset != "" {
		b.WriteString(" offset ")
		b.WriteString(offset)
	}

	b.WriteString(";")
	return b.String(), nil
}

func makeRetrieveCommand(req *http.Request) (string, error) {
	return makeSelectClause(
		req.URL.Query().Get("fields"),
		chi.URLParam(req, "setName"),
		req.URL.Query().Get("cond"),
		req.URL.Query().Get("filter"),
		req.URL.Query().Get("tag"),
		req.URL.Query().Get("omitTag"),
		req.URL.Query().Get("sort"),
		req.URL.Query().Get("limit"),
		req.URL.Query().Get("offset"),
	)
}

// Specify the JSON encoding.

type FieldDescription struct {
	Name string `json:"name"`
	// No other elements yet, but use a structure for future expansion
}

type DataRow struct {
	Values []any `json:"values"`
	// No other elements yet, but use a structure for future expansion
}

type RetrieveResponse struct {
	Status  string             `json:"status"`
	Fields  []FieldDescription `json:"fields"`
	Data    []DataRow          `json:"data"`
	Message string             `json:"message"`
}

// Translate from CCMS's API into structures with JSON encoding instructions
func ccms2local(rr *ccms.Response) RetrieveResponse {
	r := readResults(rr)[0]
	localFields := make([]FieldDescription, len(r.Fields()))
	for i, val := range r.Fields() {
		localFields[i].Name = val.Name()
	}

	localData := make([]DataRow, 0)
	for val := range r.Data() {
		values := val.Values()
		row := DataRow{Values: make([]any, len(values))}
		copy(row.Values, values)
		localData = append(localData, row)
	}

	return RetrieveResponse{
		Status:  r.Status(),
		Fields:  localFields,
		Data:    localData,
		Message: r.Message(),
	}
}

func (server *ModCyclopsServer) handleRetrieve(w http.ResponseWriter, req *http.Request, caption string) error {
	command, err := makeRetrieveCommand(req)
	if err != nil {
		return fmt.Errorf("could not make retrieve command: %w", err)
	}
	server.Log("command", command)

	resp, err := server.sendToCCMS(caption+" "+chi.URLParam(req, "setName"), command)
	if err != nil {
		return fmt.Errorf("could not retrieve: %w", err)
	}

	localrr := ccms2local(resp)
	return respondWithJSON(w, localrr, caption)
}

// -----------------------------------------------------------------------------

func (server *ModCyclopsServer) handleDropSet(w http.ResponseWriter, req *http.Request, caption string) error {
	command := "drop set " + chi.URLParam(req, "setName") + ";"
	server.Log("command", command)

	resp, err := server.sendToCCMS(caption+" "+chi.URLParam(req, "setName"), command)
	if err != nil {
		return err
	}

	localrr := ccms2local(resp)
	return respondWithJSON(w, localrr, caption)
}

// -----------------------------------------------------------------------------

func (server *ModCyclopsServer) handleAddRemoveObjects(w http.ResponseWriter, req *http.Request, caption string) error {
	// It seems weird to just shrug and say "fine" for anything posted, but for now it will suffice.
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// -----------------------------------------------------------------------------

func (server *ModCyclopsServer) handleAddRemoveTags(w http.ResponseWriter, req *http.Request, caption string) error {
	// It seems weird to just shrug and say "fine" for anything posted, but for now it will suffice.
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// -----------------------------------------------------------------------------

func unmarshalBody[T any](req *http.Request, data *T) error {
	b, err := io.ReadAll(req.Body)
	if err != nil {
		return fmt.Errorf("could not read HTTP request body: %w", err)
	}

	err = json.Unmarshal(b, &data)
	if err != nil {
		return fmt.Errorf("could not deserialize JSON from body: %w", err)
	}

	return nil
}

func (server *ModCyclopsServer) sendToCCMS(caption string, command string) (*ccms.Response, error) {
	resp, err := server.ccmsClient.Send(command)
	if err != nil {
		return nil, fmt.Errorf("could not %s: %w", caption, err)
	}
	result := readResults(resp)[0]
	if result.Status() == "error" {
		return nil, fmt.Errorf("%s failed: %s", caption, result.Message())
	}
	return resp, nil
}

func respondWithJSON(w http.ResponseWriter, data any, caption string) error {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("could not encode JSON for %s: %w", caption, err)
	}

	w.Header().Set("Content-Type", "application/json")

	// If w.write fails there is no way to report this to the client: see MODREP-37.
	_, _ = w.Write(b)
	return nil
}

func readResults(resp *ccms.Response) []ccms.Result {
	results := make([]ccms.Result, 0)
	for r := range resp.Results() {
		results = append(results, r)
	}
	return results
}
