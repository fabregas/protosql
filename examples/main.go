package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/fabregas/protosql"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
	const (
		host     = "127.0.0.1"
		port     = 5432
		user     = "postgres"
		password = "pwd"
		dbname   = "test"
	)

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	logrus.SetLevel(logrus.DebugLevel)

	logger := logrus.WithField("obg", "repo")

	r := protosql.NewRepo(db, "projects", &Project{}, logger)

	obj := &Project{
		Id:          36522,
		Name:        "test",
		Status:      ProjectStatus_PROJECT_STATUS_ACTIVE,
		Description: "test project",
		CreateTime:  timestamppb.New(time.Now()),
	}

	err = r.Insert(context.Background(), obj)
	if err != nil {
		panic(err)
	}

	obj.Status = ProjectStatus_PROJECT_STATUS_BLOCKED
	err = r.UpdateByID(context.Background(), obj)
	if err != nil {
		panic(err)
	}

	var p Project
	err = r.GetByID(context.Background(), &p, 6522)
	if err != nil {
		panic(err)
	}
	fmt.Println("PROJECT: ", &p)

	var ret []*Project
	err = r.Select(context.Background()).Where(
		protosql.NewFilter().
			NotEmptyStr("name").
			Eq("name", "test"),
	).Fetch(&ret)
	if err != nil {
		panic(err)
	}
	for _, v := range ret {
		fmt.Println(v)
	}

	fmt.Println("============================")
	ret = ret[:0]
	err = r.Select(context.Background()).
		As("p").
		LeftJoin("partner as pt", "pt.id = p.partner_id").
		Where(
			protosql.NewFilter().
				NotEmptyStr("p.name").
				Eq("pt.name", "test"),
		).
		Paginate(protosql.Page(0, 10)).
		Fetch(&ret)
	if err != nil {
		panic(err)
	}
	for _, v := range ret {
		fmt.Println(v)
	}

	fmt.Println("============================")

	type XX struct {
		protosql.DummyModel
		ProjectDescr string `db:"name=p.description"`
		PartnerName  string `db:"name=pt.name"`
	}
	var xx []*XX
	err = r.SelectCustom(
		context.Background(),
		"SELECT p.description, pt.name FROM projects as p LEFT JOIN partner as pt ON pt.id = p.partner_id",
	).
		Where(
			protosql.NewFilter().
				NotEmptyStr("p.name").
				Contain("p.name", "es").
				Eq("pt.name", "test"),
		).
		Fetch(&xx)
	if err != nil {
		panic(err)
	}

	for _, v := range xx {
		fmt.Printf("%+v\n", v)
	}
}
