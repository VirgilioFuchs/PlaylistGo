import type { RefObject } from "react";
import { useId, useState } from "react";
import { Button } from "./Button";
import { Input } from "./Input";

type PlaylistFormProps = {
  onCreate: (name: string) => void;
  busy?: boolean;
  inputRef?: RefObject<HTMLInputElement | null>;
};

export function PlaylistForm({ onCreate, busy = false, inputRef }: PlaylistFormProps) {
  const [name, setName] = useState("");
  const inputId = useId();

  return (
    <form
      className="playlist-form"
      onSubmit={(event) => {
        event.preventDefault();
        onCreate(name);
        setName("");
      }}
    >
      <Input
          id={inputId}
          ref={inputRef}
          label="Criar playlist"
          type="text"
          value={name}
          onChange={(event) => setName(event.target.value)}
          placeholder="Nome da playlist"
          autoComplete="off"
          disabled={busy}
      />
      <Button type="submit" loading={busy} loadingLabel="Criando...">Criar</Button>
    </form>
  );
}
