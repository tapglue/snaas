module Action exposing (Msg(..))

import Http
import Navigation exposing (Location)
import RemoteData exposing (WebData)
import Time exposing (Time)
import App.Model exposing (App)
import Member.Model exposing (Member)
import Route exposing (Route)
import Rule.Model exposing (Rule)
import User.Model exposing (User)


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
    | RuleDelete (Result Http.Error ())
    | Tick Time
    | TokenPersist String
    | UserFetch (WebData User)
    | UserSearch (WebData (List User))
    | UserSearchFormBlur String
    | UserSearchFormClear
    | UserSearchFormFocus String
    | UserSearchFormSubmit
    | UserSearchFormUpdate String String
    | UserUpdate (WebData User)
    | UserUpdateFormBlur String
    | UserUpdateFormClear
    | UserUpdateFormFocus String
    | UserUpdateFormSubmit
    | UserUpdateFormUpdate String String
