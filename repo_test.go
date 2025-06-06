package protosql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"google.golang.org/protobuf/types/known/durationpb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

type dbTestFunc func(t *testing.T, db *sql.DB, mock sqlmock.Sqlmock)

func withDBMock(t *testing.T, f dbTestFunc) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	f(t, db, mock)
}

func wrapTest(f dbTestFunc) func(t *testing.T) {
	return func(t *testing.T) {
		withDBMock(t, f)
	}
}

var testModel *TestModel = &TestModel{
	Id:          123,
	Name:        "test model",
	Website:     "test.com",
	Description: "model for testing Repo",
	Status:      ModelStatus_STATUS_BLOCKED,
	CreateTime:  timestamppb.Now(),
	UpdateTime:  timestamppb.Now(),
	Count:       30005000,
	Nested: &NestedModel{
		Num:    323,
		Name:   "Nested obj",
		Active: true,
	},
	NestedList: []*NestedModel{
		{
			Num:  12,
			Name: "Item in nested list",
		},
	},
	Tags:        []string{"test", "model"},
	Blob:        []byte(`123`),
	OldStatuses: []TestModelStatus{ModelStatus_STATUS_INITIAL, ModelStatus_STATUS_ACTIVE},
}

type gtTime struct {
	exp time.Time
}

func timeGreaterThan(t time.Time) gtTime {
	return gtTime{exp: t}
}

func (t gtTime) Match(v driver.Value) bool {
	ct, ok := v.(time.Time)
	if !ok {
		return false
	}
	if ct.After(t.exp) {
		return true
	}
	return false
}

func insertTest(t *testing.T, db *sql.DB, mock sqlmock.Sqlmock) {
	m := *testModel

	nestedJson, _ := json.Marshal(m.Nested)
	nestedListJson, _ := json.Marshal(m.NestedList)
	mock.ExpectExec("INSERT INTO xxx_table").WithArgs(
		m.Id,
		m.Name,
		m.Website,
		m.Description,
		m.Status,
		timeGreaterThan(m.CreateTime.AsTime()),
		timeGreaterThan(m.UpdateTime.AsTime()),
		m.OnlineDuration.AsDuration(),
		m.Count,
		nestedJson,
		pq.Array(m.Tags),
		nestedListJson,
		m.Blob,
		pq.Array(m.OldStatuses),
	).WillReturnError(nil).WillReturnResult(sqlmock.NewResult(0, 1))

	r := NewRepo(db, "xxx_table", &TestModel{}, dummyLogger{})
	err := r.Insert(context.Background(), &m)
	if err != nil {
		t.Errorf("Insert() failed: %s", err)
	}
}

func updateByIDTest(t *testing.T, db *sql.DB, mock sqlmock.Sqlmock) {
	m := *testModel

	nestedJson, _ := json.Marshal(m.Nested)
	nestedListJson, _ := json.Marshal(m.NestedList)
	mock.ExpectExec("UPDATE xxx_table").WithArgs(
		m.Id,
		m.Name,
		m.Website,
		m.Description,
		m.Status,
		m.CreateTime.AsTime(),
		timeGreaterThan(m.UpdateTime.AsTime()),
		m.OnlineDuration.AsDuration(),
		m.Count,
		nestedJson,
		pq.Array(m.Tags),
		nestedListJson,
		m.Blob,
		pq.Array(m.OldStatuses),
	).WillReturnError(nil).WillReturnResult(sqlmock.NewResult(0, 1))

	r := NewRepo(db, "xxx_table", &TestModel{}, dummyLogger{})
	err := r.UpdateByID(context.Background(), &m)
	if err != nil {
		t.Errorf("UpdateByID() failed: %s", err)
	}
}

func updateTest(t *testing.T, db *sql.DB, mock sqlmock.Sqlmock) {
	m := *testModel

	nestedJson, _ := json.Marshal(m.Nested)
	nestedListJson, _ := json.Marshal(m.NestedList)
	mock.ExpectExec("UPDATE xxx_table").WithArgs(
		m.Id,
		m.Name,
		m.Website,
		m.Description,
		m.Status,
		m.CreateTime.AsTime(),
		timeGreaterThan(m.UpdateTime.AsTime()),
		m.OnlineDuration.AsDuration(),
		m.Count,
		nestedJson,
		pq.Array(m.Tags),
		nestedListJson,
		m.Blob,
		pq.Array(m.OldStatuses),
		m.Name,
	).WillReturnError(nil).WillReturnResult(sqlmock.NewResult(0, 1))

	r := NewRepo(db, "xxx_table", &TestModel{}, dummyLogger{})
	err := r.Update(context.Background(), &m, NewFilter().Eq("name", "test model"))
	if err != nil {
		t.Errorf("Update() failed: %s", err)
	}
}

func getTest(t *testing.T, db *sql.DB, mock sqlmock.Sqlmock) {
	t0 := time.Now().Add(-time.Minute)
	t1 := time.Now()
	tags := []string{"test", "model"}
	rows := sqlmock.NewRows(
		[]string{"id", "name", "website", "descr", "status", "create_time", "update_time", "online_duration", "count", "nested", "tags", "nested_list", "blob", "old_statuses"},
	).AddRow(
		22, "test", "test.com", "some descr", 1, t0, t1, 10000, 334, `{"num": 123, "name": "some name", "active": true}`, pq.Array(&tags),
		`[{"num": 12, "name": "Item in nested list", "active": false}]`, []byte(`123`), pq.Array(testModel.OldStatuses),
	)

	mock.ExpectQuery("^SELECT (.+) FROM xxx_table").WithArgs(22).WillReturnError(nil).WillReturnRows(rows)

	r := NewRepo(db, "xxx_table", &TestModel{}, dummyLogger{})
	ret := TestModel{}
	err := r.FindByID(context.Background(), 22).FetchOne(&ret)
	if err != nil {
		t.Fatalf("FindByID() failed: %s", err)
	}

	expectEq(t, ret.Id, int32(22))
	expectEq(t, ret.Name, "test")
	expectEq(t, ret.Website, "test.com")
	expectEq(t, ret.Description, "some descr")
	expectEq(t, ret.Status, ModelStatus_STATUS_INITIAL)
	expectEq(t, ret.CreateTime.AsTime(), t0.UTC())
	expectEq(t, ret.UpdateTime.AsTime(), t1.UTC())
	expectEq(t, ret.Count, int64(334))
	expectEq(t, ret.Nested.Num, int32(123))
	expectEq(t, ret.Nested.Name, "some name")
	expectEq(t, ret.Nested.Active, true)
	expectEq(t, ret.Tags, tags)
	expectEq(t, ret.OldStatuses, testModel.OldStatuses)
}

func filterTest(t *testing.T, db *sql.DB, mock sqlmock.Sqlmock) {
	t0 := time.Now().Add(-time.Minute)
	t1 := time.Now()
	tags := []string{"test", "model"}
	rows := sqlmock.NewRows(
		[]string{"id", "name", "website", "descr", "status", "create_time", "update_time", "online_duration", "count", "nested", "tags", "nested_list", "blob", "old_statuses"},
	).AddRow(
		22, "test", "test.com", "some descr", 1, t0, t1, 10000, 334, `{"num": 123, "name": "some name", "active": true}`, pq.Array(&tags),
		`[{"num": 12, "name": "Item in nested list", "active": false}]`, []byte(`123`), pq.Array(testModel.OldStatuses),
	)

	mock.ExpectQuery("^SELECT (.+) FROM xxx_table").WithArgs(22, 1, "ololo", 32, `{"active": true}`).WillReturnError(nil).WillReturnRows(rows)

	r := NewRepo(db, "xxx_table", &TestModel{}, dummyLogger{})
	retLst := []*TestModel{}
	filter := NewFilter().Eq("id", 22).Or(NewFilter().Eq("status", 1).Eq("name", "ololo")).Neq("count", 32).Raw("1 = 1").JsonContain("nested", `{"active": true}`)
	err := r.Select(context.Background()).Where(filter).Fetch(&retLst)
	if err != nil {
		t.Fatalf("Select() failed: %s", err)
	}

	if len(retLst) != 1 {
		t.Fatal("Select() should ret 1 row")
	}
	ret := retLst[0]
	expectEq(t, ret.Id, int32(22))
	expectEq(t, ret.Name, "test")
	expectEq(t, ret.Website, "test.com")
	expectEq(t, ret.Description, "some descr")
	expectEq(t, ret.Status, ModelStatus_STATUS_INITIAL)
	expectEq(t, ret.CreateTime.AsTime(), t0.UTC())
	expectEq(t, ret.UpdateTime.AsTime(), t1.UTC())
	expectEq(t, ret.Count, int64(334))
	expectEq(t, ret.Nested.Num, int32(123))
	expectEq(t, ret.Nested.Name, "some name")
	expectEq(t, ret.Nested.Active, true)
	expectEq(t, ret.Tags, tags)
	expectEq(t, ret.OldStatuses, testModel.OldStatuses)
}

func txTest(t *testing.T, db *sql.DB, mock sqlmock.Sqlmock) {
	m := *testModel

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO xxx_table").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE xxx_table").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	r := NewRepo(db, "xxx_table", &TestModel{}, dummyLogger{})

	err := r.Transaction(context.Background(), func(ctx context.Context) error {
		err := r.Insert(ctx, &m)
		if err != nil {
			return err
		}

		err = r.UpdateByID(ctx, &m)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		t.Errorf("Transaction() failed: %s", err)
	}
}

func expectEq(t *testing.T, v1, v2 interface{}) {
	if !reflect.DeepEqual(v1, v2) {
		t.Errorf("%+v != %+v", v1, v2)
	}
}

func TestRepo(t *testing.T) {
	t.Run("insert", wrapTest(insertTest))
	t.Run("updateByID", wrapTest(updateByIDTest))
	t.Run("update", wrapTest(updateTest))
	t.Run("getbyid", wrapTest(getTest))
	t.Run("filter", wrapTest(filterTest))
	t.Run("transaction", wrapTest(txTest))
}

// dummy logger

type dummyLogger struct{}

func (l dummyLogger) Debugf(format string, args ...interface{}) { fmt.Printf(format+"\n", args...) }
func (l dummyLogger) Infof(format string, args ...interface{})  {}
func (l dummyLogger) Errorf(format string, args ...interface{}) {}

// --- model

type NestedModel struct {
	Num    int32  `protobuf:"varint,1,opt,name=num,proto3" json:"num,omitempty"`
	Name   string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Active bool   `protobuf:"bytes,3,opt,name=active,proto3" json:"active,omitempty"`
}

type TestModel struct {
	Id             int32                  `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Name           string                 `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Website        string                 `protobuf:"bytes,3,opt,name=website,proto3" json:"website,omitempty"`
	Description    string                 `protobuf:"bytes,4,opt,name=description,proto3" json:"description,omitempty"`
	Status         TestModelStatus        `protobuf:"varint,5,opt,name=status,proto3,enum=some.v1.TestModelStatus" json:"status,omitempty"`
	CreateTime     *timestamppb.Timestamp `protobuf:"bytes,6,opt,name=create_time,json=createTime,proto3" json:"create_time,omitempty"`
	UpdateTime     *timestamppb.Timestamp `protobuf:"bytes,7,opt,name=update_time,json=updateTime,proto3" json:"update_time,omitempty"`
	OnlineDuration *durationpb.Duration   `protobuf:"bytes,8,opt,name=online_duration,json=updateTime,proto3" json:"online_duration,omitempty"`
	Count          int64                  `protobuf:"bytes,9,opt,name=count,json=count,proto3" json:"count,omitempty"`
	Nested         *NestedModel           `protobuf:"bytes,10,opt,name=nested,json=nested,proto3" json:"nested,omitempty"`
	Tags           []string               `protobuf:"bytes,11,opt,name=tags,json=tags,proto3" json:"tags,omitempty"`
	NestedList     []*NestedModel         `protobuf:"bytes,12,opt,name=nested_list,json=nested_list,proto3" json:"nested_list,omitempty"`
	Blob           []byte                 `protobuf:"bytes,13,opt,name=blob,json=blob,proto3" json:"blob,omitempty"`
	OldStatuses    []TestModelStatus      `protobuf:"bytes,14,opt,name=old_statuses,json=ols_statuses,proto3" json:"old_statuses,omitempty"`
}

func (*TestModel) Reset()        {}
func (*TestModel) ProtoMessage() {}

type TestModelStatus int32

const (
	ModelStatus_STATUS_UNDEFINED TestModelStatus = 0
	ModelStatus_STATUS_INITIAL   TestModelStatus = 1
	ModelStatus_STATUS_ACTIVE    TestModelStatus = 2
	ModelStatus_STATUS_BLOCKED   TestModelStatus = 3
)

func (x TestModelStatus) Enum() *TestModelStatus {
	p := new(TestModelStatus)
	*p = x
	return p
}
