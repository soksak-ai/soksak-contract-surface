# soksak-contract-surface — 사이드카가 그리는 터미널 표면

> 번역본. 정본은 [SPEC.md](SPEC.md) 이며 이 문서는 독립 규칙을 정의하지 않는다.

계약 id: **`soksak-spec-sidecar-surface`**, 버전 **0.0.4**.

렌더 사이드카는 터미널 그리드 미러를 소유하고 그것을 칠한다. 애플리케이션은 창과 입력 장치를
소유한다. 이 계약은 그 사이의 이음매다: 사이드카가 채우는 IOSurface 링, 링과 프레임 신호를 나르는
mach 채널, 그리고 버전 있는 control envelope 위의 `surface.*` 명령. 이 저장소는 아무것도 구현하지
않고 아무것도 배포하지 않는다.

## 1. 목적과 소유

- **렌더 사이드카(픽셀 소유):** 그리드 미러, 글리프 래스터화, Metal 파이프라인, IOSurface 링, damage
  추적, preedit·선택·hover 오버레이, 셀 치수. 애플리케이션이 준 픽셀 상자에서 `cols`/`rows` 를
  계산한다.
- **애플리케이션(창 소유):** 링의 표면 하나를 레이어 트리에 합성하고, 키보드·IME·마우스·스크롤을
  받아 입력 바이트를 PTY 로 전달하고, 기하를 보고하고, 파킹용 픽셀을 읽는다. 셀도 글리프도
  아틀라스도 갖지 않는다.

프레임은 셀의 형태로 이 이음매를 넘지 않는다. 넘는 것은 링마다 한 번의 표면 identity(mach 포트)와
칠한 프레임마다 하나의 고정 크기 신호뿐이다.

## 2. 채널

- 애플리케이션은 설치본마다 서비스 이름 하나를 도출한다: `<identifier>.surface`. `<identifier>` 는
  control socket 을 이름 짓는 그 설치 식별자다. 이름은 bootstrap 의 128 바이트 한계 안이어야 하며,
  넘는 식별자는 이름과 함께 거부한다.
  사이드카 프로세스는 자기 identifier 를 갖지 않는다 — `surface.open` 이 그것을 싣고, 양쪽이 같은
  이름을 그 값에서 도출한다.
- 애플리케이션이 그 이름을 `bootstrap_check_in`(수신 권리)하고, 사이드카가 `bootstrap_look_up`(송신
  권리)한다. 자식은 자기를 spawn 한 프로세스 — 애플리케이션의 사이드카 호스트 — 의 bootstrap
  네임스페이스를 물려받는다.
- 사이드카가 보내는 첫 메시지는 `hello` 이고, `hello` 는 응답 포트를 싣는다. 애플리케이션에서
  사이드카로 가는 모든 채널 메시지는 그 응답 포트로 가므로, `hello` 이후 채널은 양방향이고 아무것도
  폴링하지 않는다.
- 채널은 darwin 에 존재한다. 다른 플랫폼은 이름과 함께 실패한다. Windows(DXGI 공유 핸들)·
  Linux(dmabuf) 채널은 백엔드가 생길 때 자기 절로 추가된다.

## 3. 채널 메시지

### mach 패킷

- 패킷의 페이로드는 정확히 와이어 메시지 하나다 — 어떤 접두도 없다. 권리가 없는 메시지는 body 워드를
  싣지 않으며 바이트가 mach 헤더 바로 뒤에서 시작한다. 권리가 있으면 complex: body, 디스크립터, 바이트 순.
- mach 크기는 4바이트 정렬이므로 수신자는 와이어 자신의 길이(`WireLength`/`wire_length`)로 패딩을
  잘라낸 뒤 해독한다.

모든 메시지는 고정 헤더로 시작한다: `magic u32 'sksf'`, `version u8 = 1`, `kind u8`,
`payloadLen u16`, 전부 big-endian. magic 이나 version 이 다른 메시지는 이름과 함께 거부한다.
pane 키는 `len u8` + UTF-8 로 이동하며 터미널 플러그인 계약의 pane 키 문법을 따른다.

| kind | 방향 | payload |
| --- | --- | --- |
| 1 hello | 사이드카 → 앱 | `sidecarIdLen u8, sidecarId`, mach 송신 권리 하나(응답 포트) |
| 2 ring | 사이드카 → 앱 | pane, `pixelW u32, pixelH u32, scale f64, cellW f64, cellH f64`, IOSurface 송신 권리 셋, 링 순서 0..2 |
| 3 frameReady | 사이드카 → 앱 | pane, `ringIndex u8, seq u64, cursorRow u16, cursorCol u16, cursorVisible u8, damageCount u8`, damage 사각형 `x,y,w,h u16` 각각 |
| 4 released | 앱 → 사이드카 | pane, `ringIndex u8` |
| 5 gap | 사이드카 → 앱 | pane — 미러가 source 연속성을 잃음; 복구는 터미널 계약을 따른다 |
| 6 ended | 사이드카 → 앱 | pane, `reasonLen u8, reason` |

## 4. 링 규칙

pane 마다 표면 셋. 각각은 정확히 한 상태에 있다: `free`, `rendering`, `signaled`, `displayed`.

- 사이드카는 `free` 표면에만 그리고, `frameReady` 로 `signaled` 로 옮기며, 애플리케이션이 해제하기
  전까지 다시 건드리지 않는다.
- 애플리케이션은 가장 최근의 `signaled` 표면을 표시하고, 직전에 표시하던 표면을 해제하며
  (`released`), 지금 표시 중인 것은 해제하지 않는다.
- `resize` 와 재연결은 링을 무효로 만든다: 사이드카가 새 `ring` 메시지를 보내고 애플리케이션은 옛
  인덱스 전부를 해제한다. 옛 표면은 링과 함께 사라지며 수명은 사이드카의 것이다.

이 상태기계는 계약의 일부다. 애플리케이션이 해제하지 않은 표면에 그리는 구현은 적합성에서
실패한다.

## 5. control envelope 위의 명령 (애플리케이션 → 사이드카)

payload 는 다른 모든 사이드카 명령처럼 `args.request` 의 JSON, 답은 `result.data`. 빠졌거나 형태가
틀린 필드는 이름과 함께 거부한다.

| 명령 | 요청 | 답 |
| --- | --- | --- |
| `surface.open` | `identifier, window, pane, pixelW, pixelH, scale, font{family, pt}, theme{fg, bg, cursor, cursorAccent, selectionBg, selectionFg, ansi[256]}, cwd?` | `cols, rows, cellW, cellH` — 링은 채널로 뒤따른다 |
| `surface.resize` | `pane, pixelW, pixelH, scale` | `cols, rows` |
| `surface.setPaused` | `pane, paused` | `{}` — paused 동안 프레임 없음 |
| `surface.preedit` | `pane, text, caret` | `{}` — 오버레이로만 그리고 PTY 에 쓰지 않는다 |
| `surface.selection` | `window, pane, action: "read"` \| `action: "clear"` \| `action: "gesture", gestureId, phase, kind, point{row,col,side}, modifiers{shift,alt,control,meta}` | 완전한 `SelectionSnapshot` |
| `surface.hover` | `pane, row, col` 또는 `pane, clear: true` | `{}` — 링크 밑줄 |
| `surface.scroll` | `pane, offset` \| `lines` \| `edge: "top"\|"bottom"` | `offset, historySize` |
| `surface.read` | `pane, lines?` | `text` — 현재 offset 의 viewport |
| `surface.theme` | `pane, theme{…}` | `{}` — 링 재생성 없음 |
| `surface.close` | `pane` | `{}` — `ended` 로 링을 끝낸다 |

`cols`/`rows` 는 사이드카의 답이지 애플리케이션의 추측이 아니다. 폰트를 측정한 쪽이 사이드카다.
애플리케이션은 답을 받은 숫자로 `pty.resize` 를 구동한다.

`surface.selection`은 strict discriminated union이다. `window`와 `pane`은 selection owner 주소이며
모든 action에서 필수다. `phase`는 `begin|update|end`, `kind`는
`simple|block|semantic|line|extend`, `side`는 `left|right`다. Point는 현재 표시된 viewport 좌표이며
render owner가 현재 scroll offset을 통해 row를 변환한다. 모든 gesture request는 modifier 네 개와 비어
있지 않은 opaque `gestureId`를 모두 전달한다. Begin이 owner를 claim하고 다른 owner의 update/end는
`STALE_GESTURE`로 거부한다. 더 늦은 begin은 이전 owner를 대체하며 clear는 조건 없이 적용한다.

모든 action은 다음 `SelectionSnapshot`을 반환한다.

```
active, text, kind, anchor{row,col,side}, focus{row,col,side}, gestureId, sequence
```

`sequence`는 clear를 포함해 pane별 단조 증가한다. Caller는 현재 관측보다 낡지 않은 snapshot만
채택한다. 비활성이면 text는 비고 kind, anchor, focus, gestureId는 null이다. Gesture 확장, 선택 text,
painter가 쓰는 row range는 engine이 소유한다. Core, application service, Plugin은 cell에서 selection을
재구성하지 않는다.

## 6. 존재

그리는 프로세스가 표시하는 프로세스다. 따라서 렌더 사이드카는 자기 PTY observer 등록에
`displays: true` 를 선언하고, PTY 데몬은 표시 중인 observer 를 방치 판정의 renderer 존재로 센다.
필드 자체는 PTY 계약에 있다. 이 절은 누가 왜 그것을 켜는지를 기술한다.

## 7. 판정

- 적합성(이 저장소): 채널 이름 도출과 길이 거부; 모든 kind 의 메시지 왕복; 해제되지 않은 표면에
  그리기를 거부하는 링 상태기계; 첫 빠진 필드를 이름 짓는 명령 payload 검증.
- 숫자(애플리케이션 쪽): `surface.composition` worst 0, `frameReady`→commit 이 한 프레임 안, paused
  동안 채널 메시지 0.
