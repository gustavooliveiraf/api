package store

import (
    "encoding/json"
    "io"
    "io/ioutil"
    "log"
    "fmt"
    "net/http"
    "strings"
    "strconv"

    "github.com/gorilla/mux"
    "github.com/gorilla/context"
    "github.com/dgrijalva/jwt-go"
)

type Controller struct {
    Repository Repository
}


func AuthenticationMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
        authorizationHeader := req.Header.Get("authorization")
        if authorizationHeader != "" {
            bearerToken := strings.Split(authorizationHeader, " ")
            if len(bearerToken) == 2 {
                token, error := jwt.Parse(bearerToken[1], func(token *jwt.Token) (interface{}, error) {
                    if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                        return nil, fmt.Errorf("There was an error")
                    }
                    return []byte("secret"), nil
                })
                if error != nil {
                    json.NewEncoder(w).Encode(Exception{Message: error.Error()})
                    return
                }
                if token.Valid {
                    log.Println("TOKEN WAS VALID")
                    context.Set(req, "decoded", token.Claims)
                    next(w, req)
                } else {
                    json.NewEncoder(w).Encode(Exception{Message: "Invalid authorization token"})
                }
            }
        } else {
            json.NewEncoder(w).Encode(Exception{Message: "An authorization header is required"})
        }
    })
}

func (c *Controller) GetToken(w http.ResponseWriter, req *http.Request) {
    var user User
    _ = json.NewDecoder(req.Body).Decode(&user)
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "username": user.Username,
        "password": user.Password,
    })

    log.Println("Username: " + user.Username);
    log.Println("Password: " + user.Password);

    tokenString, error := token.SignedString([]byte("secret"))
    if error != nil {
        fmt.Println(error)
    }
    json.NewEncoder(w).Encode(JwtToken{Token: tokenString})
}

func (c *Controller) Index(w http.ResponseWriter, r *http.Request) {
    products := c.Repository.GetProducts() // lista todos os produtos
    // log.Println(products)
    data, _ := json.Marshal(products)
    w.Header().Set("Content-Type", "application/json; charset=UTF-8")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.WriteHeader(http.StatusOK)
    w.Write(data)
    return
}

func (c *Controller) AddProduct(w http.ResponseWriter, r *http.Request) {
    var product Product
    body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576)) // read the body of the request
    
    log.Println(body)

    if err != nil {
        log.Fatalln("Error AddProduct", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    if err := r.Body.Close(); err != nil {
        log.Fatalln("Error AddProduct", err)
    }

    if err := json.Unmarshal(body, &product); err != nil {
        w.WriteHeader(422)
        log.Println(err)
        if err := json.NewEncoder(w).Encode(err); err != nil {
            log.Fatalln("Error AddProduct unmarshalling data", err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
    }

    log.Println(product)
    success := c.Repository.AddProduct(product)
    if !success {
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json; charset=UTF-8")
    w.WriteHeader(http.StatusCreated)
    return
}

func (c *Controller) SearchProduct(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    log.Println(vars)

    query := vars["query"]
    log.Println("Search Query - " + query);

    products := c.Repository.GetProductsByString(query)
    data, _ := json.Marshal(products)

    w.Header().Set("Content-Type", "application/json; charset=UTF-8")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.WriteHeader(http.StatusOK)
    w.Write(data)
    return
}

func (c *Controller) UpdateProduct(w http.ResponseWriter, r *http.Request) {
    var product Product
    body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576)) // read the body of the request
    if err != nil {
        log.Fatalln("Error UpdateProduct", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    if err := r.Body.Close(); err != nil {
        log.Fatalln("Error UpdateProduct", err)
    }

    if err := json.Unmarshal(body, &product); err != nil {
        w.Header().Set("Content-Type", "application/json; charset=UTF-8")
        w.WriteHeader(422)
        if err := json.NewEncoder(w).Encode(err); err != nil {
            log.Fatalln("Error UpdateProduct unmarshalling data", err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
    }

    log.Println(product.ID)
    success := c.Repository.UpdateProduct(product)
    
    if !success {
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json; charset=UTF-8")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.WriteHeader(http.StatusOK)
    return
}

func (c *Controller) GetProduct(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    log.Println(vars)

    id := vars["id"]
    log.Println(id);

    productid, err := strconv.Atoi(id);

    if err != nil {
        log.Fatalln("Error GetProduct", err)
    }

    product := c.Repository.GetProductById(productid)
    data, _ := json.Marshal(product)

    w.Header().Set("Content-Type", "application/json; charset=UTF-8")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.WriteHeader(http.StatusOK)
    w.Write(data)
    return
}

func (c *Controller) DeleteProduct(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    log.Println(vars)
    id := vars["id"]
    log.Println(id);

    productid, err := strconv.Atoi(id);

    if err != nil {
        log.Fatalln("Error GetProduct", err)
    }

    if err := c.Repository.DeleteProduct(productid); err != "" {
        log.Println(err);
        if strings.Contains(err, "404") {
            w.WriteHeader(http.StatusNotFound)
        } else if strings.Contains(err, "500") {
            w.WriteHeader(http.StatusInternalServerError)
        }
        return
    }
    w.Header().Set("Content-Type", "application/json; charset=UTF-8")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.WriteHeader(http.StatusOK)
    return
}