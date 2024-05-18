// Lets get a Pony to say something fun.
//
// Cow's talking, that is so last week... Ponies talking, well that is so today!
package main

import (
	"context"
	"math/rand"
)

// Borrowed from here: https://eu.usatoday.com/story/life/2023/11/30/positive-quotes-to-inspire/11359498002/
var quotes = []string{
	`"It takes courage to grow up and become who you really are." — E.E. Cummings`,
	`"Your self-worth is determined by you. You don't have to depend on someone telling you who you are." — Beyoncé`,
	`"Nothing is impossible. The word itself says 'I'm possible!'" — Audrey Hepburn`,
	`"Keep your face always toward the sunshine, and shadows will fall behind you." — Walt Whitman`,
	`"You have brains in your head. You have feet in your shoes. You can steer yourself any direction you choose. You're on your own. And you know what you know. And you are the guy who'll decide where to go." — Dr. Seuss`,
	`"Attitude is a little thing that makes a big difference." — Winston Churchill`,
	`"To bring about change, you must not be afraid to take the first step. We will fail when we fail to try." — Rosa Parks`,
	`"All our dreams can come true, if we have the courage to pursue them." — Walt Disney`,
	`"Don't sit down and wait for the opportunities to come. Get up and make them." — Madam C.J. Walker`,
	`"Champions keep playing until they get it right." — Billie Jean King`,
	`"I am lucky that whatever fear I have inside me, my desire to win is always stronger." — Serena Williams`,
	`"You are never too old to set another goal or to dream a new dream." — C.S. Lewis`,
	`"It is during our darkest moments that we must focus to see the light." — Aristotle`,
	`"Believe you can and you're halfway there." — Theodore Roosevelt`,
	`"Life shrinks or expands in proportion to one’s courage." — Anaïs Nin`,
	`"Just don't give up trying to do what you really want to do. Where there is love and inspiration, I don't think you can go wrong." — Ella Fitzgerald`,
	`"Try to be a rainbow in someone's cloud." — Maya Angelou`,
	`"If you don't like the road you're walking, start paving another one." — Dolly Parton`,
	`"Real change, enduring change, happens one step at a time." — Ruth Bader Ginsburg`,
	`"All dreams are within reach. All you have to do is keep moving towards them." — Viola Davis`,
	`"It is never too late to be what you might have been." — George Eliot`,
	`"When you put love out in the world it travels, and it can touch people and reach people in ways that we never even expected." — Laverne Cox`,
	`"Give light and people will find the way." — Ella Baker`,
	`"It always seems impossible until it's done." — Nelson Mandela`,
	`"Don’t count the days, make the days count." — Muhammad Ali`,
	`"If you risk nothing, then you risk everything." — Geena Davis`,
	`"Definitions belong to the definers, not the defined." — Toni Morrison`,
	`"When you have a dream, you've got to grab it and never let go." — Carol Burnett`,
	`"Never allow a person to tell you no who doesn’t have the power to say yes." — Eleanor Roosevelt`,
	`"When it comes to luck, you make your own." — Bruce Springsteen`,
	`"If you're having fun, that's when the best memories are built." — Simone Biles`,
	`"Failure is the condiment that gives success its flavor." — Truman Capote`,
	`"Hard things will happen to us. We will recover. We will learn from it. We will grow more resilient because of it." — Taylor Swift`,
	`"Your story is what you have, what you will always have. It is something to own." — Michelle Obama`,
	`"To live is the rarest thing in the world. Most people just exist." — Oscar Wilde`,
	`"You define beauty yourself, society doesn’t define your beauty." — Lady Gaga`,
	`"Optimism is a happiness magnet. If you stay positive, good things and good people will be drawn to you." — Mary Lou Retton`,
	`"You just gotta keep going and fighting for everything, and one day you’ll get to where you want." — Naomi Osaka`,
	`"If you prioritize yourself, you are going to save yourself." — Gabrielle Union`,
	`"No matter how far away from yourself you may have strayed, there is always a path back. You already know who you are and how to fulfill your destiny." — Oprah Winfrey`,
	`"A problem is a chance for you to do your best." — Duke Ellington`,
	`"You can’t turn back the clock. But you can wind it up again." — Bonnie Prudden`,
	`"When you can’t find someone to follow, you have to find a way to lead by example." — Roxane Gay`,
	`"There is no better compass than compassion." — Amanda Gorman`,
	`"Stand before the people you fear and speak your mind – even if your voice shakes." — Maggie Kuhn`,
	`"It’s a toxic desire to try to be perfect. I realized later in life that the challenge is not to be perfect. It’s to be whole." — Jane Fonda`,
	`"Vitality shows not only in the ability to persist but in the ability to start over." — F. Scott Fitzgerald`,
	`"The most common way people give up their power is by thinking they don’t have any." — Alice Walker`,
	`"Love yourself first and everything else falls into line." — Lucille Ball`,
	`"In three words I can sum up everything I've learned about life: It goes on." — Robert Frost`,
}

// Ponysay Dagger module
type Ponysay struct {
	// +private
	Base *Container
}

func New() *Ponysay {
	return &Ponysay{
		Base: dag.Container().From("mpepping/ponysay"),
	}
}

// Forgot cows! Lets get a Pony to say something instead
func (p *Ponysay) Say(
	ctx context.Context,
	// give the pony something fun to say
	// +optional
	// +default="Dagger is Awesome!"
	msg string,
) (string, error) {
	return p.Base.
		WithExec([]string{msg}).
		Stdout(ctx)
}

// Need an inspirational quote. These ponies have got you covered.
func (p *Ponysay) InspireMe(ctx context.Context) (string, error) {
	num := rand.Intn(len(quotes))

	return p.Base.
		WithExec([]string{quotes[num]}).
		Stdout(ctx)
}
