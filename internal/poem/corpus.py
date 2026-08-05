"""Regenerate corpus.go from 全唐詩 on wikisource, which is public domain.

    python corpus.py

The definition of what belongs is mechanical, so nobody has to have taste
about it:

    a 绝句 is four 句 of uniform length, five or seven characters each.

Everything else here is defence against the source. 全唐詩 卷520-527 were
transcribed by different hands and use at least three layouts — bare <p>
paragraphs under an <h1>, <div class="poem"> blocks under an <h3>, and poem
titles promoted to <h2>. Within one volume some poems carry ，。 and some carry
none at all, some lines end in a footnote marker, and 校勘 notes are set inline
inside the verse. So the parser does not care how a poem is laid out: it
collects the 句 however they arrive and then counts.

Three earlier versions of this were wrong, and all three were quiet about it:

  - splitting an N首 section into fours on the strength of its title turned
    every 律诗 into two 绝句 Du Mu never wrote;
  - requiring two 句 to a line dropped every unpunctuated poem, 《山行》 among
    them;
  - turning every tag into a line break cut a 句 into pieces wherever a variant
    was marked, so 《遣怀》 and 《赤壁》 could not even be found by searching.

The check at the bottom — named poems that must be present, 律诗 lines that
must not — caught all three, and it runs before anything is written.
"""

import html
import json
import os
import re
import sys
import time
import urllib.request

VOLUMES = range(520, 528)
SPLIT = re.compile(r"[，。？！；、]")
HAN = re.compile(r"^[一-鿿]+$")
CN = "一二三四五六七八九十"
AUTHOR_HEADING = re.compile(r"^杜牧([（(].*?[）)])?$")

HERE = os.path.dirname(os.path.abspath(__file__))
CACHE = os.path.join(HERE, "cache")
OUT = os.path.join(HERE, "corpus.go")
API = ("https://zh.wikisource.org/w/api.php?action=parse&page="
       "%E5%85%A8%E5%94%90%E8%A9%A9/%E5%8D%B7{}"
       "&prop=text&variant=zh-hans&format=json&formatversion=2")


def page_of(vol):
    """One rendered volume, fetched once and kept.

    zh-hans is the site's own conversion; the volumes are transcribed in a mix
    of simplified and traditional and would otherwise arrive in both.
    """
    path = os.path.join(CACHE, "%d.html" % vol)
    if os.path.exists(path) and os.path.getsize(path) > 1000:
        return open(path).read()
    req = urllib.request.Request(API.format(vol), headers={
        "User-Agent": "poemtea-corpus/1.0 (+https://github.com/jc-244/POEM-TEA)"})
    for _ in range(5):
        try:
            text = json.load(urllib.request.urlopen(req, timeout=40))["parse"]["text"]
            os.makedirs(CACHE, exist_ok=True)
            open(path, "w").write(text)
            time.sleep(3)
            return text
        except Exception as exc:
            print("  retrying 卷%d: %s" % (vol, exc), file=sys.stderr)
            time.sleep(8)
    raise SystemExit("could not fetch 卷%d" % vol)


def clean(fragment):
    """The visible text of an HTML fragment.

    Two things have to go before the tags do. 全唐詩 carries its 校勘 inline —
    落<span class="variant-text">魄<span class="variant-tooltip">一作“托”</span></span>
    — where the outer span holds the reading and the inner one the note, so
    dropping the note keeps the base text while dropping the whole span would
    lose a character. Footnote markers go the same way.

    And a tag breaks the line only if it is a block tag. Turning every tag into
    a newline cuts a 句 into pieces wherever a variant is marked.
    """
    f = re.sub(r"<sup[^>]*class=\"reference\".*?</sup>", "", fragment, flags=re.S)
    f = re.sub(r"<span[^>]*class=\"mw-editsection\".*?</span></span>", "", f, flags=re.S)
    f = re.sub(r"<span[^>]*class=\"variant-tooltip\"[^>]*>.*?</span>", "", f, flags=re.S)
    f = re.sub(r"</?(br|p|div|h[1-6]|li|tr)\b[^>]*>", "\n", f)
    f = re.sub(r"<[^>]+>", "", f)
    return html.unescape(f)


def headline(fragment):
    return " ".join(re.sub(r"\[?\s*编辑\s*\]?", "", clean(fragment).strip()).split())


def ju_of(fragment):
    """Every 句 in a fragment, however the transcriber laid it out."""
    out = []
    for line in clean(fragment).split("\n"):
        line = line.strip().replace("　", "")
        if not line:
            continue
        parts = [p.strip() for p in SPLIT.split(line)] if SPLIT.search(line) else [line]
        out += [p for p in parts if p]
    return out


def quatrain(ju):
    """The four 句 of a 绝句, or None.

    Exactly four, and no cleverness beyond that. A title saying 二首 or 六首
    tells you a section holds several poems; it does not tell you they are
    绝句, and 《长安杂题长句六首》 is six 律诗. Where several poems share a
    title the source keeps them in separate blocks, which poems_in already
    returns separately; where it does not, they are lost. Losing a real poem is
    recoverable, shipping an invented one is not.
    """
    if len(ju) != 4 or not all(HAN.match(j) for j in ju):
        return None
    n = len(ju[0])
    if n not in (5, 7) or any(len(j) != n for j in ju):
        return None
    return ju


def sections(page):
    """(title, body html) for every poem, splitting on any heading."""
    title, out = "", []
    for chunk in re.split(r"(<h[1-4][^>]*>.*?</h[1-4]>)", page, flags=re.S):
        if re.match(r"<h[1-4]", chunk):
            head = headline(chunk)
            if AUTHOR_HEADING.match(head):      # 杜牧（五）— an author, not a poem
                continue
            title = head
        elif title:
            out.append((title, chunk))
    return out


def poems_in(body):
    """One entry per poem. A <div class="poem"> is one poem; where a volume has
    none, the whole section is one poem."""
    divs = re.findall(r"<div[^>]*class=\"poem\"[^>]*>.*?</div>", body, flags=re.S)
    return [ju_of(d) for d in divs] if divs else [ju_of(body)]


def collect():
    found, seen = [], set()
    for vol in VOLUMES:
        page = page_of(vol)
        if "杜牧" not in page:
            continue
        groups = []
        for title, body in sections(page):
            name = re.sub(r"（.*?）|\(.*?\)|〈.*?〉|《.*?》", "", title).strip()
            for ju in poems_in(body):
                q = quatrain(ju)
                if not q:
                    continue
                if groups and groups[-1][0] == name:
                    groups[-1][1].append(q)
                else:
                    groups.append((name, [q]))
        for name, qs in groups:
            for i, q in enumerate(qs):
                key = "".join(q)
                if key in seen:                 # 全唐詩 repeats a few poems
                    continue
                seen.add(key)
                nth = CN[i] if i < len(CN) else str(i + 1)
                found.append({"title": name + ("·其" + nth if len(qs) > 1 else ""),
                              "ju": q})
    return found


# Named poems that must survive whatever the parser does, and 律诗 lines that
# must not. The readings are 全唐詩's own: 红烛 not 银烛, 沈沙 not 沉沙,
# 十三馀 not 十三余, 尊前 not 樽前.
MUST_HAVE = ["远上寒山石径斜", "千里莺啼绿映红", "红烛秋光冷画屏",
             "青山隐隐水迢迢", "十年一觉扬州梦", "春风十里扬州路",
             "欲把一麾江海去", "烟笼寒水月笼沙", "折戟沈沙铁未销",
             "娉娉袅袅十三馀", "多情却似总无情"]
MUST_NOT = ["一谒征南最少年", "滕阁中春绮席开", "晴日登攀好"]

HEADER = """// Code generated from 全唐詩 by corpus.py. DO NOT EDIT.
//
// 杜牧 (803-852), every 绝句 in 全唐詩 卷520-527 — four 句 of one length, five
// or seven characters each. They are set two 句 to a line, the way they are
// printed, so a whole poem is two lines.
//
// The readings are 全唐詩's own and are not always the familiar ones: 红烛 for
// 银烛, 折戟沈沙 for 沉沙, 十三馀 for 十三余, 尊前 for 樽前. Where the source
// marks a variant inline the base reading is kept and the note dropped.

package poem

var Corpus = []Poem{
"""


def main():
    found = collect()
    five = sum(1 for p in found if len(p["ju"][0]) == 5)
    print("绝句 %d 首 (五绝 %d, 七绝 %d)" % (len(found), five, len(found) - five))

    ok = True
    for want in MUST_HAVE:
        hit = next((p for p in found if any(want in j for j in p["ju"])), None)
        ok &= bool(hit)
        print(("  OK       " if hit else "  MISSING  ") + want +
              ("  -> " + hit["title"] if hit else ""))
    for bad in MUST_NOT:
        hit = next((p for p in found if any(bad in j for j in p["ju"])), None)
        ok &= not hit
        print(("  !LEAKED  " if hit else "  excluded ") + bad)
    if not ok:
        raise SystemExit("checks failed; corpus.go not written")

    with open(OUT, "w") as f:
        f.write(HEADER)
        for p in found:
            a, b, c, d = p["ju"]
            f.write("\t{\n")
            f.write('\t\tLines:  []string{"%s，%s。", "%s，%s。"},\n' % (a, b, c, d))
            f.write('\t\tAuthor: "杜牧",\n')
            f.write('\t\tSource: "%s",\n' % p["title"])
            f.write("\t},\n")
        f.write("}\n")
    print("\nwrote %s (%d poems)" % (OUT, len(found)))


main()
