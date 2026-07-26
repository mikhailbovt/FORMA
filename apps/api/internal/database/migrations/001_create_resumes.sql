CREATE TABLE resumes (
    id uuid PRIMARY KEY,
    title text NOT NULL CHECK (char_length(title) BETWEEN 1 AND 200),
    document jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT resumes_document_object CHECK (jsonb_typeof(document) = 'object')
);

CREATE INDEX resumes_updated_at_idx ON resumes (updated_at DESC, id DESC);

