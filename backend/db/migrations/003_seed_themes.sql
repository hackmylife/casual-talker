-- +goose Up
INSERT INTO courses (id, title, description, sort_order) VALUES
  ('00000000-0000-0000-0000-000000000001', '日常会話コース', '日常的な会話を通じて基本的な英語力を身につけます', 1);

INSERT INTO themes (course_id, title, description, target_phrases, base_vocabulary, sort_order) VALUES
  ('00000000-0000-0000-0000-000000000001', 'Greetings', 'あいさつ', '["Hello!", "How are you?", "Nice to meet you."]', '["hello", "good", "morning", "fine", "thanks"]', 1),
  ('00000000-0000-0000-0000-000000000001', 'Self Introduction', '自己紹介', '["My name is...", "I am from...", "I like..."]', '["name", "from", "live", "work", "hobby"]', 2),
  ('00000000-0000-0000-0000-000000000001', 'Family', '家族', '["I have...", "My mother is...", "We live together."]', '["family", "mother", "father", "sister", "brother"]', 3),
  ('00000000-0000-0000-0000-000000000001', 'Hobbies', '趣味', '["I enjoy...", "In my free time...", "I started..."]', '["hobby", "enjoy", "play", "watch", "read"]', 4),
  ('00000000-0000-0000-0000-000000000001', 'Food', '食べ物', '["I like eating...", "My favorite food is...", "I often cook..."]', '["food", "eat", "cook", "delicious", "restaurant"]', 5),
  ('00000000-0000-0000-0000-000000000001', 'Weekend', '週末', '["Last weekend I...", "On weekends I usually...", "I went to..."]', '["weekend", "Saturday", "Sunday", "relax", "went"]', 6),
  ('00000000-0000-0000-0000-000000000001', 'Shopping', '買い物', '["I want to buy...", "How much is...?", "I am looking for..."]', '["shop", "buy", "price", "store", "sale"]', 7),
  ('00000000-0000-0000-0000-000000000001', 'Weather', '天気', '["It is sunny today.", "I like rainy days.", "The weather is..."]', '["weather", "sunny", "rainy", "cold", "hot"]', 8);

-- +goose Down
DELETE FROM themes WHERE course_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM courses WHERE id = '00000000-0000-0000-0000-000000000001';
