import { useId } from "react";
import { Button } from "./Button";
import { Input } from "./Input";

type SearchFormProps = {
  value: string;
  loading?: boolean;
  onChange: (value: string) => void;
  onSubmit: (query: string) => void;
};

export function SearchForm({ value, loading = false, onChange, onSubmit }: SearchFormProps) {
  const inputId = useId();

  return (
    <form
      className="search-form"
      onSubmit={(event) => {
        event.preventDefault();
        onSubmit(value);
      }}
    >
      <Input
          id={inputId}
          label="Buscar"
          type="search"
          value={value}
          onChange={(event) => onChange(event.target.value)}
          placeholder="Busque por musica, artista ou video"
          autoComplete="off"
          spellCheck={false}
      />
      <Button type="submit" loading={loading} loadingLabel="Pesquisando...">Buscar</Button>
    </form>
  );
}
