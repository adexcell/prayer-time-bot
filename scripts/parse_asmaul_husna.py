#!/usr/bin/env python3
"""
Парсер Прекрасных имён Аллаха (Асмауль-Хусна) с сайта koran.center.
Извлекает:
- Номер и имя (транслитерация на русском и английском)
- Написание на арабском языке
- Значение (краткий перевод и полное богословское разъяснение)
- Связанные имена
- Ссылки на Коран (количество упоминаний, список сура:аят, цитаты)
- Ссылки на хадисы (если упоминаются в комментариях)
- Векторные SVG изображения арабского написания и каллиграфические PNG
"""

import os
import re
import json
import urllib.request
import urllib.error
import unicodedata

HEADERS = {'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)'}

OUTPUT_DIR = os.path.abspath(os.path.join(os.path.dirname(__file__), '..', 'data', 'asmaul_husna'))
IMAGES_DIR = os.path.join(OUTPUT_DIR, 'images')

os.makedirs(IMAGES_DIR, exist_ok=True)

def fetch_url(url: str, timeout: int = 15) -> str:
    req = urllib.request.Request(url, headers=HEADERS)
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return resp.read().decode('utf-8')

def download_file(url: str, dest_path: str, timeout: int = 15) -> bool:
    try:
        req = urllib.request.Request(url, headers=HEADERS)
        with urllib.request.urlopen(req, timeout=timeout) as resp, open(dest_path, 'wb') as f:
            f.write(resp.read())
        return True
    except Exception as e:
        print(f"Warning downloading {url}: {e}")
        return False

def slugify(text: str) -> str:
    text = text.lower().strip()
    text = re.sub(r'[\'’‘`ʻ]', '', text)
    text = re.sub(r'[\s_]+', '_', text)
    text = re.sub(r'[^a-z0-9_-]', '', text)
    return text or 'name'

def clean_arabic(text: str) -> str:
    """
    Нормализует арабский текст:
    - Заменяет персидские символы (ی, ک) на стандартные арабские (ي, ك)
    - Убирает татвиль (кашиду ـ), смещающую надстрочный алиф
    - Убирает лишние пробелы и применяет каноническую нормализацию Unicode (NFC)
    """
    text = text.replace('\u06cc', '\u064a')  # Persian yeh -> Arabic yeh
    text = text.replace('\u06a9', '\u0643')  # Persian keheh -> Arabic kaf
    text = text.replace('ـٰ', 'ٰ')
    text = text.replace('ـ', '')
    text = text.strip()
    return unicodedata.normalize('NFC', text)

def generate_svg_image(arabic_text: str, transliteration: str, translation: str, out_path: str):
    """
    Создаёт красивую векторную SVG карточку с арабским написанием имени.
    """
    cleaned_ar = clean_arabic(arabic_text)
    svg_content = f"""<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 600 320" width="600" height="320">
  <defs>
    <linearGradient id="bg-grad" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#064e3b" />
      <stop offset="50%" stop-color="#047857" />
      <stop offset="100%" stop-color="#065f46" />
    </linearGradient>
    <linearGradient id="gold-grad" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#fef08a" />
      <stop offset="100%" stop-color="#facc15" />
    </linearGradient>
    <style>
      @import url('https://fonts.googleapis.com/css2?family=Amiri:wght@700&amp;family=Inter:wght@400;600&amp;display=swap');
      .arabic {{
        font-family: 'Amiri', 'Traditional Arabic', 'Scheherazade New', 'Noto Naskh Arabic', serif;
        font-size: 80px;
        fill: url(#gold-grad);
        text-anchor: middle;
        direction: rtl;
        unicode-bidi: isolate;
        dominant-baseline: central;
        font-feature-settings: "liga" 1, "calt" 1, "mark" 1, "mkmk" 1;
        font-variant-ligatures: contextual common-ligatures;
      }}
      .translit {{
        font-family: 'Inter', system-ui, -apple-system, sans-serif;
        font-size: 26px;
        font-weight: 600;
        fill: #ffffff;
        text-anchor: middle;
      }}
      .translation {{
        font-family: 'Inter', system-ui, -apple-system, sans-serif;
        font-size: 18px;
        font-weight: 400;
        fill: #a7f3d0;
        text-anchor: middle;
      }}
      .frame {{
        stroke: #34d399;
        stroke-opacity: 0.35;
        stroke-width: 1.5;
        fill: none;
      }}
    </style>
  </defs>

  <!-- Background -->
  <rect width="100%" height="100%" rx="28" fill="url(#bg-grad)" />

  <!-- Decorative Inner Frame -->
  <rect x="14" y="14" width="572" height="292" rx="20" class="frame" />

  <!-- Arabic Calligraphy / Typography -->
  <text x="300" y="125" class="arabic" xml:lang="ar" direction="rtl">{cleaned_ar}</text>

  <!-- Transliteration and Translation -->
  <text x="300" y="225" class="translit">{transliteration}</text>
  <text x="300" y="265" class="translation">{translation}</text>
</svg>"""
    with open(out_path, 'w', encoding='utf-8') as f:
        f.write(svg_content)

def main():
    print("=== Запуск парсинга имён Аллаха с сайта koran.center ===")
    
    # 1. Получаем бандл koran.center
    html = fetch_url("https://koran.center/asmaul-husna")
    
    app_js_match = re.search(r'src=["\'](/assets/app-[^"\']+\.js)["\']', html)
    if not app_js_match:
        raise ValueError("Не удалось найти путь к assets/app-*.js в HTML")

    app_js_url = f"https://koran.center{app_js_match.group(1)}"
    print(f"Загрузка app.js: {app_js_url}")
    app_js = fetch_url(app_js_url)

    id_js_match = re.search(r'assets/(_id_-[a-zA-Z0-9_-]+\.js)', app_js)
    if not id_js_match:
        raise ValueError("Не удалось найти путь к assets/_id_-*.js в app.js")

    id_js_url = f"https://koran.center/assets/{id_js_match.group(1)}"
    print(f"Загрузка _id_.js: {id_js_url}")
    id_js = fetch_url(id_js_url)

    # 2. Извлекаем данные из app.js
    idx_do = app_js.find('Do=JSON.parse(')
    start_str = app_js.find('`', idx_do)
    end_str = app_js.find('`', start_str + 1)
    data_do = json.loads(app_js[start_str + 1 : end_str])
    do_map = {int(item['id']): item for item in data_do}

    eo_match = re.search(r'Eo\s*=\s*\[\s*\[\s*`1`.*?\]\]', app_js)
    if not eo_match:
        raise ValueError("Не найден массив русских имён Eo в app.js")

    eo_raw = eo_match.group(0)
    eo_items = re.findall(r'\[`(\d+)`,\s*`([^`]+)`,\s*`([^`]+)`,\s*`([^`]+)`\]', eo_raw)
    eo_map = {int(item[0]): {
        'id': int(item[0]),
        'arabic': item[1],
        'translation': item[2],
        'transliteration': item[3]
    } for item in eo_items}

    # 3. Извлекаем маппинг файлов со статьями из _id_.js
    ru_file_matches = dict(re.findall(r'assets/(\d+)\.ru-([a-zA-Z0-9_-]+)\.js', id_js))

    # 4. Загружаем дополнительную базу каллиграфии (Suleeyman open-source dataset)
    print("Загрузка дополнительной базы каллиграфии...")
    try:
        esma_json_url = "https://raw.githubusercontent.com/Suleeyman/api-esmaul-husna/main/assets/esmaul-husna.json"
        esma_data = json.loads(fetch_url(esma_json_url))
    except Exception as e:
        print(f"Предупреждение: не удалось получить esmaul-husna.json: {e}")
        esma_data = []

    def normalize_ar(s: str) -> str:
        s = s.replace('\u06cc', '\u064a').replace('\u06a9', '\u0643')
        s = re.sub(r'[\u064B-\u065F\u0670\u0640]', '', s)
        s = re.sub(r'[إأآ]', 'ا', s)
        s = s.replace('ة', 'ه').replace('ى', 'ي')
        return s.strip()

    calligraphy_map = {normalize_ar(item['ar']): item for item in esma_data}

    # Вспомогательный словарь для быстрого получения данных имени по ID
    quick_names_lookup = {}
    for nid, eo in eo_map.items():
        quick_names_lookup[nid] = {
            'id': nid,
            'arabic': eo['arabic'],
            'transliteration': eo['transliteration'],
            'translation': eo['translation']
        }

    all_names = []
    total = len(eo_map)
    print(f"Парсинг {total} имён Аллаха...")

    for nid in sorted(eo_map.keys()):
        eo_item = eo_map[nid]
        do_item = do_map.get(nid, {})
        
        # Получаем статью
        hash_suffix = ru_file_matches.get(str(nid))
        article_text = ""
        quran_quotes = []
        hadith_citations = []
        inline_ayah_refs = []

        if hash_suffix:
            ru_file_url = f"https://koran.center/assets/{nid}.ru-{hash_suffix}.js"
            try:
                ru_code = fetch_url(ru_file_url)

                # Цитаты из Корана
                for m in re.finditer(r'sura:[`\'"](\d+)[`\'"],\s*aya:[`\'"](\d+)[`\'"]\},[`\'"]([^`\'"]+)[`\'"]', ru_code):
                    quran_quotes.append({
                        'surah': int(m.group(1)),
                        'ayah': int(m.group(2)),
                        'text': m.group(3)
                    })

                for m in re.finditer(r'\{surah:(\d+),\s*ayah:(\d+)\}', ru_code):
                    inline_ayah_refs.append(f"{m.group(1)}:{m.group(2)}")

                # Извлечение чистого текста
                div_match = re.search(r'return\s+[a-z0-9_$]+\(\),\s*[a-z0-9_$]+\([`\'"]div[`\'"],.*', ru_code, re.DOTALL)
                render_code = div_match.group(0) if div_match else ru_code

                raw_tokens = re.findall(r'[`\'"]([^`\'"]*)[`\'"]', render_code)
                skip_exact = {'div', 'p', 'strong', 'em', 'span', 'b', 'i', 'qrn', 'text-justify', '-1', 'null'}
                
                clean_tokens = []
                for tok in raw_tokens:
                    t_str = tok.strip()
                    if not t_str or t_str in skip_exact:
                        continue
                    if t_str.startswith('./') or t_str.startswith('__name') or t_str.startswith('class') or t_str.startswith('prose'):
                        continue
                    if re.match(r'^\d+\.ru$', t_str):
                        continue
                    clean_tokens.append(tok)

                article_text = "".join(clean_tokens).strip()
                article_text = re.sub(rf'^{nid}\.ru\s*', '', article_text)

                # Хадисы
                hadith_citations = re.findall(
                    r'\[([^\]]*(?:Бухари|Муслим|Тирмизи|Абу Дауд|Насаи|Ибн Маджа|Табарани|Ахмад|сахих|хадис|Аль-Муджам)[^\]]*)\]',
                    article_text,
                    re.IGNORECASE
                )

            except Exception as e:
                print(f"Ошибка при загрузке статьи {nid}: {e}")

        # Связанные имена
        raw_related = do_item.get('related_names', [])
        related_names_enriched = []
        for r_id_str in raw_related:
            try:
                r_id = int(r_id_str)
                if r_id in quick_names_lookup:
                    related_names_enriched.append(quick_names_lookup[r_id])
            except ValueError:
                pass

        # Изображения
        en_translit = do_item.get('transliteration') or eo_item['transliteration']
        name_slug = slugify(en_translit)
        svg_filename = f"{nid:02d}_{name_slug}.svg"
        svg_local_path = os.path.join(IMAGES_DIR, svg_filename)
        
        # Генерируем SVG карточку
        generate_svg_image(
            arabic_text=eo_item['arabic'],
            transliteration=eo_item['transliteration'],
            translation=eo_item['translation'],
            out_path=svg_local_path
        )

        # Проверяем наличие внешней каллиграфии
        norm_ar = normalize_ar(eo_item['arabic'])
        calligraphy_info = calligraphy_map.get(norm_ar)
        calligraphy_svg_url = None
        calligraphy_png_url = None

        if calligraphy_info:
            img_meta = calligraphy_info.get('image', {})
            if 'svg' in img_meta:
                calligraphy_svg_url = f"https://raw.githubusercontent.com/Suleeyman/api-esmaul-husna/main/assets/static{img_meta['svg']}"
            if 'png' in img_meta and '256x256' in img_meta['png']:
                calligraphy_png_url = f"https://raw.githubusercontent.com/Suleeyman/api-esmaul-husna/main/assets/static{img_meta['png']['256x256']}"
        elif nid == 1:
            calligraphy_svg_url = "https://upload.wikimedia.org/wikipedia/commons/2/22/Allah.svg"

        # Формируем итоговый объект
        name_entry = {
            "id": nid,
            "name": eo_item['transliteration'],
            "arabic": eo_item['arabic'],
            "translation": eo_item['translation'],
            "transliteration_en": do_item.get('transliteration', ''),
            "english_meaning": do_item.get('english', ''),
            "image": {
                "svg_card_path": f"images/{svg_filename}",
                "calligraphy_svg_url": calligraphy_svg_url,
                "calligraphy_png_url": calligraphy_png_url
            },
            "meaning": article_text,
            "related_names": related_names_enriched,
            "quran": {
                "mentions_count": do_item.get('quran_mentions', 0),
                "references": do_item.get('quran_references', []),
                "inline_references": inline_ayah_refs,
                "quotes": quran_quotes
            },
            "hadith_references": hadith_citations,
            "source_url": f"https://koran.center/asmaul-husna/{nid}"
        }

        all_names.append(name_entry)
        if nid % 20 == 0 or nid == total:
            print(f"Обработано {nid}/{total} имён...")

    # Сохраняем итоговый JSON
    json_path = os.path.join(OUTPUT_DIR, 'asmaul_husna.json')
    with open(json_path, 'w', encoding='utf-8') as f:
        json.dump(all_names, f, ensure_ascii=False, indent=2)

    print(f"\nГотово! Сохранено {len(all_names)} имён:")
    print(f"- Данные: {json_path}")
    print(f"- SVG изображения: {IMAGES_DIR}/")

if __name__ == '__main__':
    main()
