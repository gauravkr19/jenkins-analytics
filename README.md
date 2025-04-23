`docker run --name jenkins-pg \
  -e POSTGRES_USER=jenkins \
  -e POSTGRES_PASSWORD=jenkins \
  -e POSTGRES_DB=jenkins \
  -p 5432:5432 \
  -d postgres:16

User: jenkins
Password: jenkins
Database: jenkins
Port: 5432`