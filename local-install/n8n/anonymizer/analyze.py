from fastapi import FastAPI
from pydantic import BaseModel
import spacy
import re

app = FastAPI()

nlp = spacy.load("de_core_news_lg")


class InputText(BaseModel):
    text: str
    subject: str = ""


def remove_signature(text):
    patterns = [
        r"\n--\s*\n",
        r"\nMit freundlichen Grüßen.*",
        r"\nFreundliche Grüße.*",
        r"\nViele Grüße.*",
        r"\nBeste Grüße.*",
        r"\nKind regards.*"
    ]

    for pattern in patterns:
        match = re.search(pattern, text, flags=re.I | re.S)
        if match:
            return text[:match.start()].strip()

    return text


def extract_entities(text):
    doc = nlp(text)

    persons = set()

    for ent in doc.ents:
        if ent.label_ == "PER":
            persons.add(ent.text.strip())

    return {
        "persons": sorted(persons, key=len, reverse=True)
    }


def extract_regex(text):

    # E-Mail
    emails = list(set(re.findall(
        r'\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b',
        text,
        re.I
    )))

    # IBAN
    # Vor Telefon behandeln, da lange Zahlenfolgen einer IBAN
    # sonst teilweise als Telefonnummer erkannt werden können.
    ibans = list(set(re.findall(
        r'\b[A-Z]{2}\d{2}(?:[ \u00A0]?[A-Z0-9]){11,30}\b',
        text,
        re.I
    )))

    # BIC
    bics = list(set(re.findall(
        r'\b[A-Z]{4}[A-Z]{2}[A-Z0-9]{2}(?:[A-Z0-9]{3})?\b',
        text
    )))

    # Telefon
    phones = list(set(re.findall(
        r'(?:\+49\s?\(?\d+\)?[\d\s\-\/]{5,}|0\d[\d\s\-\/]{6,})',
        text
    )))

    # Straße + Hausnummer
    addresses = list(set(re.findall(
        r'\b[A-ZÄÖÜ][A-Za-zÄÖÜäöüß\s\-]+'
        r'(?:straße|str\.?|weg|platz|allee|gasse|ring|steig|pfad|ufer|chaussee|damm|bogen|promenade)'
        r'\s+\d+[a-zA-Z]?\b',
        text,
        re.I
    )))

    # PLZ + Ort
    cities = list(set(re.findall(
        r'\b\d{5}\s+[A-ZÄÖÜ][A-Za-zÄÖÜäöüß\-]+\b',
        text
    )))

    # Anreden
    greetings = list(set(re.findall(
        r'\b(?:Herr|Frau)\s+[A-ZÄÖÜ][A-Za-zÄÖÜäöüß\-]+'
        r'|(?:Hallo|Hi|Servus|Moin|Guten Tag|Liebe?r?|Sehr geehrte[rn]?|Sehr geehrter)'
        r'\s+(?:Herr|Frau)?\s*[A-ZÄÖÜ][A-Za-zÄÖÜäöüß\-]+',
        text,
        re.I
    )))

    return {
        "emails": emails,
        "phones": phones,
        "addresses": addresses,
        "cities": cities,
        "greetings": greetings,
        "ibans": ibans,
        "bics": bics
    }


def safe_replace(text, value, replacement):
    pattern = re.escape(value)
    return re.sub(pattern, replacement, text)


def anonymize_text(text, entities):

    clean = text

    # Wichtig:
    # IBAN/BIC vor Telefon ersetzen, damit Bestandteile einer IBAN
    # nicht vorher als Telefonnummer anonymisiert werden.

    for iban in sorted(entities["ibans"], key=len, reverse=True):
        clean = safe_replace(clean, iban, "[IBAN]")

    for bic in sorted(entities["bics"], key=len, reverse=True):
        clean = safe_replace(clean, bic, "[BIC]")

    for greeting in sorted(entities["greetings"], key=len, reverse=True):
        clean = safe_replace(clean, greeting, "[ANREDE]")

    for email in sorted(entities["emails"], key=len, reverse=True):
        clean = safe_replace(clean, email, "[EMAIL]")

    for phone in sorted(entities["phones"], key=len, reverse=True):
        clean = safe_replace(clean, phone, "[TELEFON]")

    for address in sorted(entities["addresses"], key=len, reverse=True):
        clean = safe_replace(clean, address, "[ADRESSE]")

    for city in sorted(entities["cities"], key=len, reverse=True):
        clean = safe_replace(clean, city, "[ORT]")

    for person in sorted(entities["persons"], key=len, reverse=True):
        clean = re.sub(
            rf'\b{re.escape(person)}\b',
            "[NAME]",
            clean
        )

    return clean


def process_text(text, remove_sig=False):

    if not text:
        return {
            "clean_text": "",
            "entities": {
                "persons": [],
                "emails": [],
                "phones": [],
                "addresses": [],
                "cities": [],
                "greetings": [],
                "ibans": [],
                "bics": []
            }
        }

    if remove_sig:
        text = remove_signature(text)

    ner_entities = extract_entities(text)
    regex_entities = extract_regex(text)

    entities = {
        **ner_entities,
        **regex_entities
    }

    clean_text = anonymize_text(text, entities)

    return {
        "clean_text": clean_text,
        "entities": entities
    }


@app.post("/analyze")
def analyze(input: InputText):

    body_result = process_text(
        input.text,
        remove_sig=True
    )

    subject_result = process_text(
        input.subject,
        remove_sig=False
    )

    return {
        "clean_text": body_result["clean_text"],
        "clean_subject": subject_result["clean_text"],
        "entities": body_result["entities"],
        "subject_entities": subject_result["entities"]
    }