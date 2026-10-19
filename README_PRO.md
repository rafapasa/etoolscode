# eToolsCode Pro

Interface em `/pro` com:

- seletor de projeto (`/api/projects` + `projeto.json` → raiz)
- lista de arquivos modificados (`/api/git/status`)
- diff lado a lado HEAD × working tree (Monaco + `/api/git/file`)
- timeline de audit (`/api/audit/recent`)

Rodar com o serviço normal (`etoolscode --ui`) e abrir `/pro`.
Não é necessário `DSAC_UI_PRO` para acessar `/pro`.
