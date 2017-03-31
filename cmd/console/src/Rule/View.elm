module Rule.View exposing (viewRule, viewRuleItem, viewRuleTable)

import Dict
import Html
    exposing
        ( Html
        , a
        , div
        , h2
        , h3
        , h4
        , pre
        , section
        , span
        , strong
        , table
        , tbody
        , td
        , text
        , th
        , thead
        , tr
        )
import Html.Attributes exposing (class, title)
import Html.Events exposing (onClick)
import Rule.Model exposing (Entity(..), Recipient, Rule, Target)


viewActivated : Bool -> Html msg
viewActivated active =
    if active then
        span [ class "nc-icon-outline ui-1_check-circle-08" ] []
    else
        span [ class "nc-icon-outline ui-1_circle-remove" ] []


viewEcosystemButton : Int -> Html msg
viewEcosystemButton ecosystem =
    case ecosystem of
        1 ->
            div [ class "icon", title "iOS" ]
                [ span [ class "nc-icon-outline design-2_apple" ] []
                , span [] [ text "iOS" ]
                ]

        _ ->
            div [ class "icon", title "Unknown" ]
                [ span [ class "nc-icon-outline ui-2_alert" ] []
                , span [] [ text "Unknown" ]
                ]

viewEcosystemIcon : Int -> Html msg
viewEcosystemIcon ecosystem =
    case ecosystem of
        1 ->
            span [ class "nc-icon-outline design-2_apple", title "iOS" ] []

        _ ->
            span [ class "nc-icon-outline ui-2_alert", title "unknown" ] []

viewEntityIcon : Entity -> Html msg
viewEntityIcon entity =
    case entity of
        Connection ->
            span [ class "nc-icon-outline arrows-2_conversion", title "Connection" ] []

        Event ->
            span [ class "nc-icon-outline ui-1_bell-53", title "event" ] []

        Object ->
            span [ class "nc-icon-outline ui-1_database", title "Object" ] []

        Reaction ->
            span [ class "nc-icon-outline ui-2_like", title "Reaction" ] []

        UnknownEntity ->
            span [ class "nc-icon-outline ui-2_alert", title "Unknown" ] []


viewEntityButton : Entity -> Html msg
viewEntityButton entity =
    let
        ( name, icon ) =
            case entity of
                Connection ->
                    ( "Connections", "arrows-2_conversion" )

                Event ->
                    ( "Events", "ui-1_bell-53" )

                Object ->
                    ( "Objects", "ui-1_database" )

                Reaction ->
                    ( "Reactions", "ui-2_like" )

                UnknownEntity ->
                    ( "Unknown", "ui-2_alert" )

    in
        div [ class "icon", title name ]
            [ span [ class ("nc-icon-outline " ++ icon) ] []
            , span [] [ text "Connections" ]
            ]

viewRecipient : Recipient -> Html msg
viewRecipient recipient =
    div [ class "recipient" ]
        [ div [ class "meta" ]
            ((List.map viewTarget recipient.targets)
                ++ [ div [ class "urn" ]
                        [ span [] [ text "URN: " ]
                        , pre [] [ text recipient.urn ]
                        ]
                   ]
            )
        , div [ class "templates" ]
            [ viewTemplates recipient.templates
            ]
        ]

viewRule : Rule -> Html msg
viewRule rule =
    div []
        [ viewRuleDescription rule
        , h4 []
            [ span [ class "icon nc-icon-outline users_mobile-contact" ] []
            , span [] [ text "Recipients" ]
            ]
        , div [ class "recipients" ] (List.map viewRecipient rule.recipients)
        ]

viewRuleDescription : Rule -> Html msg
viewRuleDescription rule =
    h3 []
        [ span [] [ text "A rule for" ]
        , viewEntityButton rule.entity
        , span [] [ text "called" ]
        , strong [] [ text rule.name ]
        , span [] [ text "targeting the" ]
        , viewEcosystemButton rule.ecosystem
        , span [] [ text "platform." ]
        ]


viewRuleItem : msg -> Rule -> Html msg
viewRuleItem msg rule =
    tr [ onClick msg ]
        [ td [ class "icon" ] [ viewActivated rule.active ]
        , td [ class "icon" ] [ viewEcosystemIcon rule.ecosystem ]
        , td [ class "icon" ] [ viewEntityIcon rule.entity ]
        , td [ class "icon" ] [ text (toString (List.length rule.recipients)) ]
        , td [] [ text rule.name ]
        ]


viewRuleTable : (Rule -> Html msg) -> List Rule -> Html msg
viewRuleTable item rules =
    let
        list =
            List.sortWith sortByEntity rules

    in
        table [ class "navigation" ]
            [ thead []
                [ tr []
                    [ th [ class "icon" ] [ text "active" ]
                    , th [ class "icon" ] [ text "ecosystem" ]
                    , th [ class "icon" ] [ text "entity" ]
                    , th [ class "icon" ] [ text "recipients" ]
                    , th [] [ text "name" ]
                    ]
                ]
            , tbody [] (List.map item list)
            ]

viewTarget : Target -> Html msg
viewTarget target =
    div [ class "target" ]
        [ span [] [ text "Target: " ]
        , strong [] [ text (Rule.Model.targetString target) ]
        ]

viewTemplate : ( String, String ) -> Html msg
viewTemplate ( lang, template ) =
    tr []
        [ td [] [ text lang ]
        , td [] [ text template ]
        ]

viewTemplates templates =
    table []
        [ thead []
            [ tr []
                [ th [] [ text "lang" ]
                , th [] [ text "template" ]
                ]
            ]
        , tbody [] (List.map viewTemplate (Dict.toList templates))
        ]



sortByEntity : Rule -> Rule -> Order
sortByEntity a b =
    case ( a.entity, b.entity ) of
        ( Connection, Connection ) ->
            EQ

        ( Connection, _ ) ->
            LT

        ( Event, Connection ) ->
            GT

        ( Event, Event ) ->
            EQ

        ( Event, _ ) ->
            LT

        ( Object, Connection ) ->
            GT

        ( Object, Event ) ->
            GT

        ( Object, Object ) ->
            EQ

        ( Object, _ ) ->
            LT

        ( Reaction, Reaction ) ->
            EQ

        ( Reaction, _ ) ->
            GT

        ( UnknownEntity, _ ) ->
            GT
