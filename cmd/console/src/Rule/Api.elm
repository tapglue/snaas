module Rule.Api exposing (activateRule, deactivateRule, deleteRule, getRule, listRules)

import Http
import RemoteData exposing (WebData, sendRequest)
import Rule.Model exposing (Rule, decode, decodeList)


returnId : String -> Http.Response String -> Result String String
returnId id response =
    if response.status.code == 204 then
        Ok id
    else
        Err response.status.message


activateRule : (Result Http.Error String -> msg) -> String -> String -> Cmd msg
activateRule msg appId ruleId =
    Http.request
        { body = Http.emptyBody
        , expect = Http.expectStringResponse (returnId ruleId)
        , headers = []
        , method = "PUT"
        , timeout = Nothing
        , url = (ruleUrl appId ruleId) ++ "/activate"
        , withCredentials = False
        }
        |> Http.send msg


deactivateRule : (Result Http.Error String -> msg) -> String -> String -> Cmd msg
deactivateRule msg appId ruleId =
    Http.request
        { body = Http.emptyBody
        , expect = Http.expectStringResponse (returnId ruleId)
        , headers = []
        , method = "PUT"
        , timeout = Nothing
        , url = (ruleUrl appId ruleId) ++ "/deactivate"
        , withCredentials = False
        }
        |> Http.send msg


deleteRule : (Result Http.Error () -> msg) -> String -> String -> Cmd msg
deleteRule msg appId ruleId =
    Http.request
        { body = Http.emptyBody
        , expect = expectEmpty
        , headers = []
        , method = "DELETE"
        , timeout = Nothing
        , url = ruleUrl appId ruleId
        , withCredentials = False
        }
        |> Http.send msg


getRule : String -> String -> Cmd (WebData Rule)
getRule appId ruleId =
    Http.get (ruleUrl appId ruleId) decode
        |> sendRequest


listRules : String -> Cmd (WebData (List Rule))
listRules appId =
    Http.get ("/api/apps/" ++ appId ++ "/rules") decodeList
        |> sendRequest


expectEmpty : Http.Expect ()
expectEmpty =
    Http.expectStringResponse readEmpty


readEmpty : Http.Response String -> Result String ()
readEmpty response =
    if response.status.code == 204 then
        Ok ()
    else
        Err response.status.message


ruleUrl : String -> String -> String
ruleUrl appId ruleId =
    "/api/apps/" ++ appId ++ "/rules/" ++ ruleId
