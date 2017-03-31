module Action exposing (Msg(..))

import Http
import Navigation exposing (Location)
import RemoteData exposing (WebData)
import Time exposing (Time)
import App.Model exposing (App)
import Member.Model exposing (Member)
import Route exposing (Route)
import Rule.Model exposing (Rule)


type Msg
    = AppFormBlur String
    | AppFormClear
    | AppFormFocus String
    | AppFormSubmit
    | AppFormUpdate String String
    | FetchApp (WebData App)
    | FetchApps (WebData (List App))
    | FetchRule (WebData Rule)
    | FetchRules (WebData (List Rule))
    | LocationChange Location
    | MemberFetch (WebData Member)
    | MemberLogin (WebData Member)
    | Navigate Route
    | NewApp (WebData App)
    | RuleActivate (Result Http.Error String)
    | RuleActivateAsk String
    | RuleDeactivate (Result Http.Error String)
    | RuleDeactivateAsk String
    | RuleDeleteAsk String
    | RuleDelete (WebData Bool)
    | Tick Time
    | TokenPersist String
