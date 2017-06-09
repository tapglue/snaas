module Rule.Model exposing (Entity(..), Recipient, Rule, Target, decode, decodeList, targetString)

import Dict exposing (Dict)
import Json.Decode as Decode


-- MODEL


type Criteria
    = EventCriteria
    | ObjectCriteria


type Entity
    = Connection
    | Event
    | Object
    | Reaction
    | UnknownEntity


type alias Recipient =
    { query : List Query
    , targets : List Target
    , templates : Dict String String
    , urn : String
    }


type alias Rule =
    { active : Bool
    , criteria : Criteria
    , deleted : Bool
    , ecosystem : Int
    , entity : Entity
    , id : String
    , name : String
    , recipients : List Recipient
    }


type Target
    = Commenters
    | PostOwner
    | UnknownTarget


type alias Query =
    ( String, String )


matchEntity : Int -> Entity
matchEntity enum =
    case enum of
        0 ->
            Connection

        1 ->
            Event

        2 ->
            Object

        3 ->
            Reaction

        _ ->
            UnknownEntity


matchTarget : Query -> Target
matchTarget query =
    case query of
        ( "objectOwner", """{ "object_ids": [ {{.Parent.ID}} ], "owned": true, "types": [ "tg_comment" ]}""" ) ->
            Commenters

        ( "parentOwner", "" ) ->
            PostOwner

        ( _, _ ) ->
            UnknownTarget


targetString : Target -> String
targetString target =
    case target of
        Commenters ->
            "Commenters"

        PostOwner ->
            "PostOwner"

        UnknownTarget ->
            "Unknown"



-- DECODERS


decode : Decode.Decoder Rule
decode =
    Decode.map8 Rule
        (Decode.field "active" Decode.bool)
        (Decode.field "criteria" decodeCriteria)
        (Decode.field "deleted" Decode.bool)
        (Decode.field "ecosystem" Decode.int)
        (Decode.andThen decodeEntity (Decode.field "entity" Decode.int))
        (Decode.field "id" Decode.string)
        (Decode.field "name" Decode.string)
        (Decode.field "recipients" (Decode.list decodeRecipient))


decodeCriteria : Decode.Decoder Criteria
decodeCriteria =
    Decode.succeed EventCriteria


decodeEntity : Int -> Decode.Decoder Entity
decodeEntity raw =
    Decode.succeed (matchEntity raw)


decodeList : Decode.Decoder (List Rule)
decodeList =
    Decode.field "rules" (Decode.list decode)


decodeRecipient : Decode.Decoder Recipient
decodeRecipient =
    Decode.map4 Recipient
        (Decode.field "query" (Decode.keyValuePairs Decode.string))
        (Decode.andThen decodeRecipientTarget (Decode.field "query" (Decode.keyValuePairs Decode.string)))
        (Decode.field "templates" (Decode.dict Decode.string))
        (Decode.field "urn" Decode.string)


decodeRecipientTarget : List ( String, String ) -> Decode.Decoder (List Target)
decodeRecipientTarget queries =
    Decode.succeed (List.map matchTarget queries)
