package main

func subquestionsParser(p *parser, v string) interface{} {
	sub := &QuestionSubquestionConfig{}

	for {
		block, eof := p.readBlock([]string{"end subquestions"}, true)
		if eof || block == nil {
			return sub
		}
		switch b := block.(type) {
		default:
			p.err("Subquestions cannot contain block of type %T", block)
		case comment:
		case *Question:
			sub.Questions = append(sub.Questions, b)
		case *Screen:
			sub.Screens = append(sub.Screens, b)
		}
	}
}
