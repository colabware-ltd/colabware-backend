
db.createUser(
    {
      user: "mongo-root",
      pwd: "CYQ9MUMnXWg3c3ypV2",
      roles: [ { role: "root", db: "admin" } ]
    }
)

db.createUser(
    {
      user: "colabware",
      pwd: "zfbj3c7oEFgsuSrTx6",
      roles: [ 
          { role: "readWrite", db: "colabware" }, ]
    }
  )

db.createUser(
{
    user: "mongo-express",
    pwd: "CiaSe7EPBsVjt",
    roles: [ 
        { role: "readWrite", db: "colabware" }, ]
}
)