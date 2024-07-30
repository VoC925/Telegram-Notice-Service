ПОМЕНЯТЬ ПРИ ДЕПЛОЕ: 
    1. путь на pathtoLogFile в логере

СДЕЛАТЬ:
    1. переделать сервер, так чтобы он принимал по запросу код авторизации
    и делал запрос на получение токена, затем токен отправлялся в сервис.
    Написать клиент для сервера.

    Логика получения токена:
    а. GET запрос
        https://oauth.yandex.ru/authorize?
        response_type=code
        & client_id=<идентификатор приложения>
    б. POST запрос
        POST /token HTTP/1.1
        Host: oauth.yandex.ru
        Content-type: application/x-www-form-urlencoded
        Content-Length: <длина тела запроса>
        [Authorization: Basic <закодированная строка client_id:client_secret>]

        grant_type=authorization_code
        & code=<код подтверждения>
        [& client_id=<идентификатор приложения>]
        [& client_secret=<пароль приложения>]
    Формат ответа:
        200 OK
        Content-type: application/json

        {
        "token_type": "bearer",
        "access_token": "AQAAAACy1C6ZAAAAfa6vDLuItEy8pg-iIpnDxIs",
        "expires_in": 124234123534,
        "refresh_token": "1:GN686QVt0mmakDd9:A4pYuW9LGk0_UnlrMIWklkAuJkUWbq27loFekJVmSYrdfzdePBy7:A-2dHOmBxiXgajnD-kYOwQ",
        "scope": "login:info login:email login:avatar"
        }