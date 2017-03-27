module Formo exposing (..)

import Dict exposing (Dict)


-- MODEL


type alias Element =
    { errors : List ValidationError
    , focused : Bool
    , validators : List ElementValidator
    , value : String
    }


type alias Elements =
    Dict String Element


type alias ElementValidator =
    String -> String -> ValidationError


type alias Form =
    { elements : Elements
    , validated : Bool
    }


type alias ValidationError =
    Maybe String


initForm : List ( String, List ElementValidator ) -> Form
initForm fields =
    let
        elements =
            List.foldl
                (\e -> Dict.insert (Tuple.first e) (initElement e))
                Dict.empty
                fields
    in
        Form elements False


initElement : ( String, List ElementValidator ) -> Element
initElement ( _, validators ) =
    Element [] False validators ""


blurElement : Form -> String -> Form
blurElement form field =
    case Dict.get field form.elements of
        Nothing ->
            form

        Just element ->
            let
                elements =
                    Dict.insert
                        field
                        { element | focused = False }
                        form.elements
            in
                { form | elements = elements }


elementErrors : Form -> String -> List String
elementErrors form field =
    let
        validate validator value =
            case (validator value) of
                Nothing ->
                    ""

                Just s ->
                    s
    in
        case Dict.get field form.elements of
            Nothing ->
                []

            Just element ->
                List.map (\v -> validate (v field) element.value) element.validators
                    |> List.filter (\e -> e /= "")


elementIsFocused : Form -> String -> Bool
elementIsFocused form field =
    case Dict.get field form.elements of
        Nothing ->
            False

        Just element ->
            element.focused


elementIsValid : Form -> String -> Bool
elementIsValid form field =
    elementErrors form field
        |> List.isEmpty


elementValue : Form -> String -> String
elementValue form field =
    case Dict.get field form.elements of
        Nothing ->
            ""

        Just element ->
            element.value


focusElement : Form -> String -> Form
focusElement form field =
    case Dict.get field form.elements of
        Nothing ->
            form

        Just element ->
            let
                elements =
                    Dict.insert
                        field
                        { element | focused = True }
                        form.elements
            in
                { form | elements = elements }


formIsValid : Form -> Bool
formIsValid form =
    Dict.toList form.elements
        |> List.map (\( k, _ ) -> elementIsValid form k)
        |> List.filter (\f -> f /= True)
        |> List.isEmpty


formIsValidated : Form -> Bool
formIsValidated form =
    form.validated


updateElementValue : Form -> String -> String -> Form
updateElementValue form field value =
    case Dict.get field form.elements of
        Nothing ->
            form

        Just element ->
            let
                elements =
                    Dict.insert
                        field
                        { element | value = value }
                        form.elements
            in
                { form | elements = elements }


validateForm : Form -> ( Form, Bool )
validateForm form =
    ( { form | validated = True }, formIsValid form )


validatorExist : String -> String -> Maybe String
validatorExist _ value =
    if String.isEmpty value then
        Just "can't be empty"
    else
        Nothing


validatorMinLength : Int -> String -> String -> Maybe String
validatorMinLength minLength _ value =
    if String.length value >= minLength then
        Nothing
    else
        Just ("must have at least " ++ toString minLength ++ " characters")


validatorMaxLength : Int -> String -> String -> Maybe String
validatorMaxLength maxLength _ value =
    if String.length value > maxLength then
        Just ("can't have more than " ++ toString maxLength ++ " characters")
    else
        Nothing
