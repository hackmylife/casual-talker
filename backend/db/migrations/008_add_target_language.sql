-- +goose Up
ALTER TABLE courses ADD COLUMN target_language TEXT NOT NULL DEFAULT 'en';

-- Rename existing English course
UPDATE courses SET target_language = 'en' WHERE id = '00000000-0000-0000-0000-000000000001';
UPDATE courses SET title = '英語 日常会話コース' WHERE id = '00000000-0000-0000-0000-000000000001';

-- Insert Italian course
INSERT INTO courses (id, title, description, target_language, sort_order) VALUES
  ('00000000-0000-0000-0000-000000000002', 'イタリア語 日常会話コース', 'イタリア語の日常会話を練習します', 'it', 2);

INSERT INTO themes (course_id, title, description, target_phrases, base_vocabulary, sort_order) VALUES
  ('00000000-0000-0000-0000-000000000002', 'Saluti', 'あいさつ', '["Ciao!", "Come stai?", "Piacere di conoscerti."]', '["ciao", "buongiorno", "bene", "grazie", "piacere"]', 1),
  ('00000000-0000-0000-0000-000000000002', 'Presentazione', '自己紹介', '["Mi chiamo...", "Sono di...", "Mi piace..."]', '["nome", "città", "lavoro", "hobby", "famiglia"]', 2),
  ('00000000-0000-0000-0000-000000000002', 'Famiglia', '家族', '["Ho una sorella.", "Mia madre è...", "Viviamo insieme."]', '["famiglia", "madre", "padre", "sorella", "fratello"]', 3),
  ('00000000-0000-0000-0000-000000000002', 'Hobby', '趣味', '["Mi piace...", "Nel tempo libero...", "Ho iniziato a..."]', '["hobby", "leggere", "cucinare", "sport", "musica"]', 4),
  ('00000000-0000-0000-0000-000000000002', 'Cibo', '食べ物', '["Mi piace mangiare...", "Il mio piatto preferito è...", "Cucino spesso..."]', '["cibo", "mangiare", "cucinare", "ristorante", "delizioso"]', 5),
  ('00000000-0000-0000-0000-000000000002', 'Fine settimana', '週末', '["Lo scorso fine settimana...", "Di solito nel weekend...", "Sono andato a..."]', '["weekend", "sabato", "domenica", "rilassarsi", "uscire"]', 6),
  ('00000000-0000-0000-0000-000000000002', 'Shopping', '買い物', '["Vorrei comprare...", "Quanto costa?", "Sto cercando..."]', '["negozio", "comprare", "prezzo", "vendita", "bello"]', 7),
  ('00000000-0000-0000-0000-000000000002', 'Tempo', '天気', '["Oggi è soleggiato.", "Mi piace la pioggia.", "Il tempo è..."]', '["tempo", "sole", "pioggia", "freddo", "caldo"]', 8);

-- Insert Korean course
INSERT INTO courses (id, title, description, target_language, sort_order) VALUES
  ('00000000-0000-0000-0000-000000000003', '韓国語 日常会話コース', '韓国語の日常会話を練習します', 'ko', 3);

INSERT INTO themes (course_id, title, description, target_phrases, base_vocabulary, sort_order) VALUES
  ('00000000-0000-0000-0000-000000000003', '인사', 'あいさつ', '["안녕하세요!", "잘 지내세요?", "만나서 반갑습니다."]', '["안녕", "좋은", "아침", "감사합니다", "반갑습니다"]', 1),
  ('00000000-0000-0000-0000-000000000003', '자기소개', '自己紹介', '["제 이름은...입니다.", "저는...에서 왔습니다.", "저는...를 좋아합니다."]', '["이름", "나라", "살다", "일하다", "취미"]', 2),
  ('00000000-0000-0000-0000-000000000003', '가족', '家族', '["저는...이/가 있습니다.", "어머니는...", "같이 살아요."]', '["가족", "어머니", "아버지", "언니/누나", "형/오빠"]', 3),
  ('00000000-0000-0000-0000-000000000003', '취미', '趣味', '["저는...를 즐깁니다.", "시간이 있을 때...", "시작했어요."]', '["취미", "읽기", "요리", "운동", "음악"]', 4),
  ('00000000-0000-0000-0000-000000000003', '음식', '食べ物', '["저는...를 좋아합니다.", "가장 좋아하는 음식은...", "자주 요리해요."]', '["음식", "먹다", "요리하다", "맛있는", "식당"]', 5),
  ('00000000-0000-0000-0000-000000000003', '주말', '週末', '["지난 주말에...", "보통 주말에는...", "...에 갔어요."]', '["주말", "토요일", "일요일", "쉬다", "나가다"]', 6),
  ('00000000-0000-0000-0000-000000000003', '쇼핑', '買い物', '["...를 사고 싶어요.", "얼마예요?", "...를 찾고 있어요."]', '["가게", "사다", "가격", "세일", "예쁜"]', 7),
  ('00000000-0000-0000-0000-000000000003', '날씨', '天気', '["오늘은 맑아요.", "비 오는 날이 좋아요.", "날씨가..."]', '["날씨", "맑은", "비", "춥다", "덥다"]', 8);

-- Insert Portuguese course
INSERT INTO courses (id, title, description, target_language, sort_order) VALUES
  ('00000000-0000-0000-0000-000000000004', 'ポルトガル語 日常会話コース', 'ポルトガル語の日常会話を練習します', 'pt', 4);

INSERT INTO themes (course_id, title, description, target_phrases, base_vocabulary, sort_order) VALUES
  ('00000000-0000-0000-0000-000000000004', 'Saudações', 'あいさつ', '["Olá!", "Como vai?", "Prazer em conhecê-lo."]', '["olá", "bom", "dia", "obrigado", "prazer"]', 1),
  ('00000000-0000-0000-0000-000000000004', 'Apresentação', '自己紹介', '["Meu nome é...", "Eu sou de...", "Eu gosto de..."]', '["nome", "cidade", "trabalho", "hobby", "família"]', 2),
  ('00000000-0000-0000-0000-000000000004', 'Família', '家族', '["Eu tenho...", "Minha mãe é...", "Moramos juntos."]', '["família", "mãe", "pai", "irmã", "irmão"]', 3),
  ('00000000-0000-0000-0000-000000000004', 'Hobbies', '趣味', '["Eu gosto de...", "No meu tempo livre...", "Eu comecei a..."]', '["hobby", "ler", "cozinhar", "esporte", "música"]', 4),
  ('00000000-0000-0000-0000-000000000004', 'Comida', '食べ物', '["Eu gosto de comer...", "Minha comida favorita é...", "Eu cozinho frequentemente..."]', '["comida", "comer", "cozinhar", "restaurante", "delicioso"]', 5),
  ('00000000-0000-0000-0000-000000000004', 'Fim de semana', '週末', '["No último fim de semana...", "Nos fins de semana eu geralmente...", "Eu fui para..."]', '["fim de semana", "sábado", "domingo", "relaxar", "sair"]', 6),
  ('00000000-0000-0000-0000-000000000004', 'Compras', '買い物', '["Eu quero comprar...", "Quanto custa?", "Estou procurando..."]', '["loja", "comprar", "preço", "venda", "bonito"]', 7),
  ('00000000-0000-0000-0000-000000000004', 'Clima', '天気', '["Hoje está ensolarado.", "Eu gosto de dias chuvosos.", "O tempo está..."]', '["tempo", "ensolarado", "chuvoso", "frio", "quente"]', 8);

-- +goose Down
DELETE FROM themes WHERE course_id IN ('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000004');
DELETE FROM courses WHERE id IN ('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000004');
ALTER TABLE courses DROP COLUMN IF EXISTS target_language;
UPDATE courses SET title = '日常会話コース' WHERE id = '00000000-0000-0000-0000-000000000001';
