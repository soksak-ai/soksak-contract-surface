# soksak-contract-surface — 사이드카가 그리는 터미널 표면

> 번역본. 정본은 [SPEC.md](SPEC.md) 이며 이 문서는 독립 규칙을 정의하지 않는다.

계약 id: **`soksak-spec-sidecar-surface`**, 버전 **0.0.7**.

렌더 사이드카는 터미널 그리드 미러를 소유하고 그것을 칠한다. 애플리케이션은 창과 입력 장치를
소유한다. 이 계약은 그 사이의 이음매다: 사이드카가 채우는 IOSurface 링, 링과 프레임 신호를 나르는
mach 채널, 그리고 버전 있는 control envelope 위의 `surface.*` 명령. 이 저장소는 아무것도 구현하지
않고 아무것도 배포하지 않는다.

## 1. 목적과 소유

- **렌더 사이드카(픽셀 소유):** 그리드 미러, 글리프 래스터화, Metal 파이프라인, IOSurface 링, damage
  추적, preedit·선택·hover 오버레이, 셀 치수. 애플리케이션이 준 픽셀 상자에서 `surface.dim` 은 서피스가 그리는 빛을 덜어냅니다. 서피스가 칠하는 모든 색에 곱해지며 뒤의 무엇에도
닿지 않습니다. 서피스는 불투명하고, dim 을 투명도로 선언하면 뒤의 문서가 화면에 나옵니다. 2026-09-04
실측 — park 된 서피스를 대신하는 그림을 그 밑에 깔자 pane 이 0.5×(191−127) 만큼 두 프레임 밝아졌고,
그림이 그려지기 전에 서피스를 내리자 두 프레임 어두워졌습니다. 불투명한 서피스에는 두 프레임이 다
없습니다.

`cols`/`rows` 를
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
| `surface.measure` | `pixelW, pixelH, scale, font{family, pt}` | `cols, rows, cellW, cellH` — pane, ring, process를 만들지 않는다 |
| `surface.open` | `identifier, window, pane, pixelW, pixelH, scale, font{family, pt}, theme{fg, bg, cursor, cursorAccent, selectionBg, selectionFg, ansi[256]}, cwd?` | `cols, rows, cellW, cellH` — 링은 채널로 뒤따른다 |
| `surface.resize` | `pane, pixelW, pixelH, scale` | `cols, rows` |
| `surface.setPaused` | `pane, paused` | `{}` — paused 동안 프레임 없음 |
| `surface.preedit` | `pane, text, caret` | `{}` — 오버레이로만 그리고 PTY 에 쓰지 않는다 |
| `surface.selection` | `window, pane, action: "read"` \| `action: "clear"` \| `action: "gesture", gestureId, phase, kind, point{row,col,side}, modifiers{shift,alt,control,meta}` | 완전한 `SelectionSnapshot` |
| `surface.hover` | `pane, row, col` 또는 `pane, clear: true` | `{}` — 링크 밑줄 |
| `surface.pointer` | `window, pane, point{x,y}, phase: "down"\|"move"\|"up", button: "none"\|"left"\|"middle"\|"right", clickCount, modifiers{shift,alt,control,meta}` | engine 결과: `route: "mouse-report"\|"ignored", dataB64` |
| `surface.wheel` | `window, pane, point{x,y}, deltaX, deltaY, deltaMode: "pixel"\|"line"\|"page", modifiers{shift,alt,control,meta}` | engine 결과: `route: "scrollback"\|"mouse-report"\|"alternate-scroll"\|"ignored", offset, historySize, dataB64` |
| `surface.focus` | `window, pane, focused` | `focused, cursorPresentation: "engine"\|"hollow-block"` |
| `surface.scroll` | `pane, offset` \| `lines` \| `edge: "top"\|"bottom"`; 양수 `lines`는 history 방향, 음수 `lines`는 bottom 방향 | `offset, historySize` |
| `surface.read` | `pane, lines?` | `text` — 현재 offset 의 viewport |
| `surface.theme` | `pane, theme{…}` | `{}` — 링 재생성 없음 |
| `surface.dim` | `window, pane, dim: 0..1` | `{}` — 다음 프레임부터 적용 |
| `surface.close` | `pane` | `{}` — `ended` 로 링을 끝낸다 |

`cols`/`rows` 는 사이드카의 답이지 애플리케이션의 추측이 아니다. 폰트를 측정한 쪽이 사이드카다.
`surface.measure`는 mirror, pane, ring, PTY를 만들지 않는다. 새 pane은 먼저 이 명령으로 측정하고 같은
숫자를 observer 준비, `pty.open`, engine 구독에 전달한다. 그 다음 `surface.open`은 같은 pixel/font
사실로 ring을 만든다. Process 시작 뒤 resize는 이후 geometry 변경에만 사용하며 새 shell의 초기 크기를
정하는 방법이 아니다.

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

`surface.wheel`은 DOM 경계에서 값을 바꾸지 않고 입력 장치의 사실을 보존합니다. `deltaMode`는 각
delta의 단위가 pixel, line, page 중 무엇인지 나타내고, `point`는 surface 기준 CSS pixel이며 네 modifier를
모두 요구합니다. 0인 delta, 유한하지 않은 좌표, 알 수 없는 field는 거부합니다. Render owner는 현재
engine mode에서 정확히 하나의 route를 선택합니다. `scrollback`은 null이 아닌 `offset`과 `historySize`,
null `dataB64`를 반환합니다. `mouse-report`와 `alternate-scroll`은 비어 있지 않은 base64 PTY input과 null
scroll 상태를 반환합니다. `ignored`는 세 effect field를 모두 null로 반환합니다. Application이 이 응답을
검증하고 decode한 input을 PTY에 쓰는 유일한 process입니다. Core와 Plugin은 mouse escape sequence를
encode하지 않습니다.

`surface.focus`는 terminal parser state가 아니라 presentation state입니다. Focused에서는 engine이
소유한 cursor shape와 blink policy를 복원합니다. Unfocused에서는 renderer animation clock을 멈추고
steady hollow block 하나를 그리되, 다시 focus될 때 돌아올 engine shape/blink 값은 보존합니다. 요청은
`window`와 `pane`에 묶이며 boolean 누락이나 focus와 모순되는 engine answer를 거부합니다. Adapter가 이
policy를 위해 DECSCUSR나 private mode 12를 다시 parse하지 않습니다.

`surface.pointer`도 surface 기준 CSS 좌표와 모든 modifier를 보존합니다. Down과 up은 물리 button 하나와
양수 click count를 요구합니다. Move는 누른 button을 지정하거나 button 없는 motion이면 `none`을
지정합니다. Render owner는 현재 1000/1002/1003 mode와 phase를 비교해 정확히 하나의 route를
반환합니다. `mouse-report`는 비어 있지 않은 base64 input을, `ignored`는 null input을 반환합니다.
Shift는 terminal mouse capture를 우회해 Plugin의 local selection을 유지합니다. Application은 반환
byte를 검증하고 wheel input과 같은 단일 PTY writer를 통해 기록합니다.

## 6. 존재

그리는 프로세스가 표시하는 프로세스다. 따라서 렌더 사이드카는 자기 PTY observer 등록에
`displays: true` 를 선언하고, PTY 데몬은 표시 중인 observer 를 방치 판정의 renderer 존재로 센다.
필드 자체는 PTY 계약에 있다. 이 절은 누가 왜 그것을 켜는지를 기술한다.

## 7. 판정

- 적합성(이 저장소): 채널 이름 도출과 길이 거부; 모든 kind 의 메시지 왕복; 해제되지 않은 표면에
  그리기를 거부하는 링 상태기계; 첫 빠진 필드를 이름 짓는 명령 payload 검증.
- 숫자(애플리케이션 쪽): `surface.composition` worst 0, `frameReady`→commit 이 한 프레임 안, paused
  동안 채널 메시지 0.
