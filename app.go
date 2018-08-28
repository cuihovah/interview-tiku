package main

import (
	"bufio"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/h2non/filetype.v1"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

type InsertQuestion struct {
	Stem      string        `json:"stem" bson:"stem"`
	Code      string        `json:"code" bson:"code"`
	Knowledge string        `json:"knowledge" bson:"knowledge"`
	Diff      int           `json:"difficulty" bson:"difficulty"`
	Image     string        `json:"image" bson:"image"`
}

type Question struct {
	Id        bson.ObjectId `json:"id" bson:"_id"`
	Stem      string        `json:"stem" bson:"stem"`
	Code      string        `json:"code" bson:"code"`
	Knowledge string        `json:"knowledge" bson:"knowledge"`
	Diff      int           `json:"difficulty" bson:"difficulty"`
	Image     string        `json:"image" bson:"image"`
}

type PutQuestion struct {
	Id        bson.ObjectId `json:"id" bson:"_id"`
	Stem      string        `json:"stem" bson:"stem"`
	Code      string        `json:"code" bson:"code"`
	Knowledge string        `json:"knowledge" bson:"knowledge"`
	Diff      int           `json:"difficulty" bson:"difficulty"`
}

type Resume struct {
	Id         bson.ObjectId `json:"id" bson:"_id"`
	Name       string        `json:"name" bson:"name"`
	Age        int           `json:"age" bson:"age"`
	Tag        string        `json:"tag" bson:"tag"`
	Level      string        `json:"level" bson:"level"`
	Knowledges []string      `json:"knowledges" bson:"knowledges"`
	Ques       interface{}   `json:"questions" bson:"questions"`
}

var session *mgo.Session

func main() {
	session, err := mgo.Dial("mongodb://cuihovah:7576583asd@47.88.49.197:27017/tiku")
	if err != nil {
		log.Fatal(err.Error())
		return
	}
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)
	router := httprouter.New()
	router.POST("/questions", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		file, _, err := r.FormFile("fileUpload")
		fname := ""
		if err == nil {
			defer file.Close()
			rd := bufio.NewReader(file)
			p := make([]byte, 5*1024*1024)
			size, _ := rd.Read(p)
			hash := fmt.Sprintf("%x", md5.Sum(p))
			fileName := string(hash[:])
			ioutil.WriteFile("./static/images/"+fileName, p[:size], 0644)
			fname = "/static/images/" + fileName
		}

		stem := r.FormValue("stem")
		code := r.FormValue("code")
		knowledge := r.FormValue("keyword")
		question := InsertQuestion{
			Stem:      stem,
			Code:      code,
			Knowledge: knowledge,
			Image:     fname,
		}
		collection := session.DB("tiku").C("question")
		err = collection.Insert(&question)
		if err != nil {
			log.Println(err.Error())
		}
		http.Redirect(w, r, "/", 302)
	})

	router.POST("/interviews", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		defer func() {
			if p := recover(); p != nil {
				msg := fmt.Sprintf("panic recover! p: %v", p)
				log.Println(msg)
			}
		}()

		body, _ := ioutil.ReadAll(r.Body)
		resume := Resume{}
		err := json.Unmarshal([]byte(body), &resume)

		if err != nil {
			panic(err.Error())
		}

		if resume.Age < 0 {
			resume.Age = -1
		}

		level := []int{1, 2, 3, 4, 5}
		if resume.Level == "初级" {
			level = []int{1, 2, 3}
		} else if resume.Level == "中级" {
			level = []int{2, 3, 4}
		} else if resume.Level == "高级" {
			level = []int{3, 4, 5}
		}

		collection := session.DB("tiku").C("question")
		questions := []Question{}
		err = collection.Find(bson.M{
			"knowledge":  bson.M{"$in": resume.Knowledges},
			"difficulty": bson.M{"$in": level},
		}).All(&questions)
		if err != nil {
			log.Fatal(err)
		}
		resume.Ques = questions
		w.Header().Set("content-type", "application/json")
		retval, _ := json.Marshal(resume)
		w.Write(retval)
	})

	router.POST("/resumes", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		body, _ := ioutil.ReadAll(r.Body)
		resume := Resume{}
		err := json.Unmarshal([]byte(body), &resume)
		if err != nil {
			log.Fatal(err.Error())
			return
		}
		if resume.Age < 0 {
			resume.Age = -1
		}
		collection := session.DB("tiku").C("resume")
		err = collection.Insert(&resume)
		if err != nil {
			log.Fatal(err)
		}
		w.Header().Set("content-type", "application/json")
		w.Write([]byte("{}"))
	})

	router.PUT("/questions/:id", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		body, _ := ioutil.ReadAll(r.Body)
		question := PutQuestion{}
		err := json.Unmarshal(body, &question)
		if err != nil {
			log.Fatal(err.Error())
			return
		}
		collection := session.DB("tiku").C("question")
		err = collection.UpdateId(bson.ObjectIdHex(ps.ByName("id")), bson.M{
			"$set": bson.M{
				"stem": question.Stem,
				"code": question.Code,
				"knowledge": question.Knowledge,
				"difficulty": question.Diff,
			},
		})
		if err != nil {
			log.Fatal(err)
		}
		w.Header().Set("content-type", "application/json")
		w.Write([]byte("{}"))
	})

	router.GET("/static/images/:name", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		content, _ := ioutil.ReadFile("./static/images/" + ps.ByName("name"))
		kind, unknow := filetype.Match(content)
		if unknow != nil {
			log.Fatal(unknow.Error())
		}
		w.Header().Set("Content-Type", kind.MIME.Value)
		w.Write(content)
	})

	router.GET("/", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		buf, _ := ioutil.ReadFile("./index.html")
		w.Header().Set("content-type", "text/html")
		w.Write(buf)
	})

	router.GET("/interview.html", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		buf, _ := ioutil.ReadFile("./interview.html")
		w.Header().Set("content-type", "text/html")
		w.Write(buf)
	})

	router.GET("/edit.html", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		buf, _ := ioutil.ReadFile("./edit.html")
		w.Header().Set("content-type", "text/html")
		w.Write(buf)
	})

	router.GET("/questions.html", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		buf, _ := ioutil.ReadFile("./questions.html")
		w.Header().Set("content-type", "text/html")
		w.Write(buf)
	})

	router.GET("/questions", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		vars := r.URL.Query()
		queryLimit := "10"
		queryOffset := "0"
		knowledge := ""
		if len(vars["limit"]) != 0 {
			queryLimit = vars["limit"][0]
		}

		if len(vars["offset"]) != 0 {
			queryOffset = vars["offset"][0]
		}

		if len(vars["knowledge"]) != 0 {
			knowledge = vars["knowledge"][0]
		}

		limit, err := strconv.Atoi(queryLimit)
		if err != nil {
			limit = 10
		}
		if limit <= 0 {
			limit = 10
		}

		offset, err := strconv.Atoi(queryOffset)
		if err != nil {
			offset = 0
		}
		if offset < 0 {
			offset = 0
		}
		cond := bson.M{}
		if knowledge != "" {
			cond["knowledge"] = knowledge
		}

		questions := []Question{}
		collection := session.DB("tiku").C("question")
		w.Header().Set("content-type", "application/json")
		err = collection.Find(cond).Skip(offset).Limit(limit).All(&questions)
		if err != nil {

			log.Println(err.Error())
		}
		retval, err := json.Marshal(questions)
		if err != nil {
			w.Write([]byte{})
		} else {
			w.Write(retval)
		}
	})

	router.GET("/questions/:id", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		collection := session.DB("tiku").C("question")
		question := Question{}
		id := ps.ByName("id")
		err = collection.Find(bson.M{"_id": bson.ObjectIdHex(id)}).One(&question)
		if err != nil {
			log.Fatal(err)
		}
		w.Header().Set("content-type", "application/json")
		retval, _ := json.Marshal(question)
		w.Write(retval)
	})

	router.GET("/knowledges", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		knowledges := []string{}
		collection := session.DB("tiku").C("question")
		err = collection.Find(bson.M{}).Distinct("knowledge", &knowledges)
		if err != nil {
			log.Fatal(err)
		}
		w.Header().Set("content-type", "application/json")
		retval, _ := json.Marshal(knowledges)
		w.Write(retval)
	})

	log.Fatal(http.ListenAndServe(":10989", router))
}
