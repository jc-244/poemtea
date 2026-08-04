// Package poem holds the corpus.
//
// COPYRIGHT RULE FOR THIS FILE — read before adding anything.
//
// Everything here must be unambiguously public domain: published in the US
// before 1929, or by an author dead more than 70 years, and — this is the one
// people get wrong — the *translation* must clear the bar too, not just the
// original. A Tang poem is public domain; a 1990s English rendering of it is
// not. Pound's Cathay (1915) is safe precisely because Pound himself is out of
// copyright, not because Li Bai is.
//
// A poem left in its own language never has that problem, because there is no
// translator to have died recently. 杜牧 died in 852 and the Chinese below is
// his own.
//
// EDITORIAL RULE — the harder one.
//
// A line qualifies only if it is an image, not an argument. "Petals on a wet,
// black bough" can be abandoned mid-read and lose nothing. An aphorism cannot:
// it asks to be finished, and anything that asks to be finished turns waiting
// back into a task. When in doubt, ask whether you could look away at the
// second line without feeling you had dropped something.
//
// Three lines is the ceiling. Longer poems are excerpted, and the excerpt has
// to stand alone without feeling truncated.
//
// A Chinese quatrain is set two 句 to a line, the way it is printed, so a whole
// one is two lines and the ceiling never binds. It has not lost half its lines
// — please do not restore it to four.
package poem

type Poem struct {
	Lines  []string
	Author string
	Source string // where the excerpt came from, if it is one
}

// Corpus is deliberately small. A few hundred is the eventual target; seeing
// the same line a third time is not a bug in a thing like this.
var Corpus = []Poem{
	{
		Lines:  []string{"The apparition of these faces in the crowd;", "Petals on a wet, black bough."},
		Author: "Ezra Pound",
		Source: "In a Station of the Metro, 1913",
	},
	{
		Lines:  []string{"The fog comes", "on little cat feet."},
		Author: "Carl Sandburg",
		Source: "Fog, 1916",
	},
	{
		Lines:  []string{"Among twenty snowy mountains,", "The only moving thing", "Was the eye of the blackbird."},
		Author: "Wallace Stevens",
		Source: "Thirteen Ways of Looking at a Blackbird, 1917",
	},
	{
		Lines:  []string{"It was evening all afternoon.", "It was snowing", "And it was going to snow."},
		Author: "Wallace Stevens",
		Source: "Thirteen Ways of Looking at a Blackbird, 1917",
	},
	{
		Lines:  []string{"so much depends", "upon", "a red wheel barrow"},
		Author: "William Carlos Williams",
		Source: "The Red Wheelbarrow, 1923",
	},
	{
		Lines:  []string{"Whirl up, sea—", "whirl your pointed pines,"},
		Author: "H. D.",
		Source: "Oread, 1914",
	},
	{
		Lines:  []string{"Listen..", "With faint dry sound,", "Like steps of passing ghosts,"},
		Author: "Adelaide Crapsey",
		Source: "November Night, 1915",
	},
	{
		Lines:  []string{"A Route of Evanescence", "With a revolving Wheel—"},
		Author: "Emily Dickinson",
	},
	{
		Lines:  []string{"I'll tell you how the Sun rose—", "A Ribbon at a time—"},
		Author: "Emily Dickinson",
	},
	{
		Lines:  []string{"There's a certain Slant of light,", "Winter Afternoons—"},
		Author: "Emily Dickinson",
	},
	{
		Lines:  []string{"Water, is taught by thirst."},
		Author: "Emily Dickinson",
	},
	{
		Lines:  []string{"Stray birds of summer come to my window", "to sing and fly away."},
		Author: "Rabindranath Tagore",
		Source: "Stray Birds, 1916",
	},
	{
		Lines:  []string{"The mist, like love, plays upon the heart of the hills", "and brings out surprises of beauty."},
		Author: "Rabindranath Tagore",
		Source: "Stray Birds, 1916",
	},
	{
		Lines:  []string{"The world puts off its mask of vastness to its lover."},
		Author: "Rabindranath Tagore",
		Source: "Stray Birds, 1916",
	},
	{
		Lines:  []string{"The paired butterflies are already yellow with August", "Over the grass in the West garden;"},
		Author: "Li Bai, tr. Ezra Pound",
		Source: "Cathay, 1915",
	},
	{
		Lines:  []string{"Greatly shining,", "The Autumn moon floats in the thin sky;"},
		Author: "Amy Lowell",
		Source: "Wind and Silver, 1919",
	},
	{
		Lines:  []string{"Night from a railroad car window", "Is a great, dark, soft thing"},
		Author: "Carl Sandburg",
		Source: "Window, 1916",
	},
	{
		Lines:  []string{"Who has seen the wind?", "Neither I nor you:"},
		Author: "Christina Rossetti",
	},
	{
		Lines:  []string{"The night is darkening round me,", "The wild winds coldly blow;"},
		Author: "Emily Brontë",
	},
	{
		Lines:  []string{"And I shall have some peace there,", "for peace comes dropping slow,"},
		Author: "W. B. Yeats",
		Source: "The Lake Isle of Innisfree, 1890",
	},
	{
		Lines:  []string{"To see a World in a Grain of Sand", "And a Heaven in a Wild Flower"},
		Author: "William Blake",
		Source: "Auguries of Innocence",
	},
	{
		Lines:  []string{"I believe a leaf of grass is no less", "than the journey-work of the stars."},
		Author: "Walt Whitman",
		Source: "Song of Myself, 1855",
	},
	{
		Lines:  []string{"A man said to the universe:", "\"Sir, I exist!\""},
		Author: "Stephen Crane",
		Source: "War Is Kind, 1899",
	},
	{
		Lines:  []string{"The sea is calm to-night.", "The tide is full, the moon lies fair"},
		Author: "Matthew Arnold",
		Source: "Dover Beach, 1867",
	},
	{
		Lines:  []string{"Season of mists and mellow fruitfulness,", "Close bosom-friend of the maturing sun;"},
		Author: "John Keats",
		Source: "To Autumn, 1820",
	},

	// 杜牧 (803–852). Set two 句 to a line — see the editorial rule above.
	{
		Lines:  []string{"远上寒山石径斜，白云生处有人家。", "停车坐爱枫林晚，霜叶红于二月花。"},
		Author: "杜牧",
		Source: "山行",
	},
	{
		Lines:  []string{"千里莺啼绿映红，水村山郭酒旗风。", "南朝四百八十寺，多少楼台烟雨中。"},
		Author: "杜牧",
		Source: "江南春",
	},
	{
		Lines:  []string{"银烛秋光冷画屏，轻罗小扇扑流萤。", "天阶夜色凉如水，卧看牵牛织女星。"},
		Author: "杜牧",
		Source: "秋夕",
	},
	{
		Lines:  []string{"青山隐隐水迢迢，秋尽江南草未凋。", "二十四桥明月夜，玉人何处教吹箫。"},
		Author: "杜牧",
		Source: "寄扬州韩绰判官",
	},
	{
		Lines:  []string{"菱透浮萍绿锦池，夏莺千啭弄蔷薇。", "尽日无人看微雨，鹭鸶时立小荷泥。"},
		Author: "杜牧",
		Source: "齐安郡后池绝句",
	},
	{
		Lines:  []string{"娉娉袅袅十三余，豆蔻梢头二月初。", "春风十里扬州路，卷上珠帘总不如。"},
		Author: "杜牧",
		Source: "赠别二首·其一",
	},
	{
		Lines:  []string{"多情却似总无情，唯觉樽前笑不成。", "蜡烛有心还惜别，替人垂泪到天明。"},
		Author: "杜牧",
		Source: "赠别二首·其二",
	},
	{
		Lines:  []string{"落魄江湖载酒行，楚腰纤细掌中轻。", "十年一觉扬州梦，赢得青楼薄幸名。"},
		Author: "杜牧",
		Source: "遣怀",
	},
	{
		Lines:  []string{"清时有味是无能，闲爱孤云静爱僧。", "欲把一麾江海去，乐游原上望昭陵。"},
		Author: "杜牧",
		Source: "将赴吴兴登乐游原一绝",
	},
	{
		// A 五律 is eight 句, so it is excerpted like any other long poem.
		Lines:  []string{"雨过一蝉噪，飘萧松桂秋。", "青苔满阶砌，白鸟故迟留。"},
		Author: "杜牧",
		Source: "题扬州禅智寺,前两联",
	},
	{
		Lines:  []string{"街垂千步柳，霞映两重城。"},
		Author: "杜牧",
		Source: "扬州三首·其一,摘句",
	},
}

// Deck hands out poems in a shuffled order and only reshuffles once the whole
// corpus has been seen. Sampling at random would let the same line come up
// twice in five minutes, which reads as broken rather than as repetition.
type Deck struct {
	order []int
	i     int
	seed  uint64
}

func NewDeck(seed uint64) *Deck {
	d := &Deck{seed: seed}
	d.reshuffle()
	return d
}

func (d *Deck) reshuffle() {
	d.order = make([]int, len(Corpus))
	for i := range d.order {
		d.order[i] = i
	}
	for i := len(d.order) - 1; i > 0; i-- {
		d.seed ^= d.seed << 13
		d.seed ^= d.seed >> 7
		d.seed ^= d.seed << 17
		j := int(d.seed % uint64(i+1))
		d.order[i], d.order[j] = d.order[j], d.order[i]
	}
	d.i = 0
}

func (d *Deck) Next() Poem {
	if d.i >= len(d.order) {
		d.reshuffle()
	}
	p := Corpus[d.order[d.i]]
	d.i++
	return p
}
