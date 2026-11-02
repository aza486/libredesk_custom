import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
const workflow = JSON.parse(readFileSync(new URL('./AI_Learning_Processor_v9.json', import.meta.url)))
const node = name => workflow.nodes.find(n => n.name === name)
test('import artifact is inactive, credential-free and fences old publisher from managed rows and facts', () => {
  assert.equal(workflow.active, false)
  assert.ok(workflow.nodes.every(n => !n.credentials))
  assert.match(node('Claim RAG Publication').parameters.query, /NOT EXISTS \(SELECT 1 FROM learning_reviews/)
  assert.match(node('Claim RAG Publication').parameters.query, /r.source_type <> 'employee_fact'/)
  assert.match(node('Finalize RAG Publication').parameters.query, /NOT EXISTS \(SELECT 1 FROM learning_reviews/)
})
test('complete original v8 guards still reject operations, finance and binding promises', () => {
  const js = node('Prepare Publication Input').parameters.jsCode
  const guard = new Function(js.slice(0, js.indexOf('const row =')) + '; return detectOperationalRisk;')()
  for (const [question, answer] of [
    ['Können Sie meine Bestellung stornieren?', 'Ja, das können wir tun.'],
    ['Wo ist meine Bestellung?', 'Wir garantieren Lieferung spätestens morgen.'],
    ['Ich möchte eine Rückzahlung', 'Wir erstatten Ihnen den Betrag.']
  ]) assert.equal(guard(question, answer, '', '', '').unsafe, true)
  assert.equal(guard('Wann ist das Büro geöffnet?', 'Samstags geschlossen.', '', '', '').unsafe, false)
})

const v10 = JSON.parse(readFileSync(new URL('./AI_Learning_Processor_v10.json', import.meta.url)))
const v10node = name => v10.nodes.find(n => n.name === name)
const comparison = overrides => ({ feedback_id:'aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa', feedback_type:'manual_resolution',manual_resolution:true, question:'An welchen Tagen ist das Büro geöffnet?', human_response:'Das Büro ist samstags geschlossen.',feedback_reason:'Bitte Öffnungstage beachten.',ai_suggestion:'',original_categories:['firmenfragen'],...overrides })
const fact = overrides => ({learning_type:'employee_fact',topic:'Öffnungstage',guidance:'Das Büro ist samstags geschlossen.',rationale:'Mitarbeiter nennt allgemeine Öffnungstage.',fact_statement:'Das Büro ist samstags geschlossen.',evidence_quote:'Das Büro ist samstags geschlossen.',category:'firmenfragen',retrieval_text:'An welchen Tagen ist das Büro geöffnet?',evidence_source:'human_answer',limitations:'Nur Büro, keine Aussage über Online-Bestellungen.',review_reason:'Allgemeine Firmeninformation unabhängig prüfen.',target_agent:'company_knowledge',...overrides})
const validate = (source, candidates, assessment='factual') => new Function('$','$input',v10node('Validate Learning Candidates').parameters.jsCode)(()=>({first:()=>({json:source})}),{first:()=>({json:{output:{feedback_assessment:assessment,candidates}}})})[0].json

test('v10 is inactive, credential-free and preserves the complete original operational guard',()=>{
 assert.equal(v10.active,false)
 assert.ok(!v10.id && v10.nodes.every(n=>!n.credentials))
 for(const name of ['Validate Learning Candidates','Prepare Publication Input']) {
  const prefix = code => code.slice(0,code.indexOf(name==='Validate Learning Candidates'?'const source=':'const row ='))
  assert.equal(prefix(v10node(name).parameters.jsCode),prefix(node(name).parameters.jsCode))
 }
 assert.match(v10node('Claim RAG Publication').parameters.query,/pipeline_version/)
 assert.match(v10node('Finalize RAG Publication').parameters.query,/vector_norm/)
})
test('general fact produces a complete inactive manual proposal with grounded evidence and scope',()=>{
 const result=validate(comparison(),[fact()]); assert.equal(result.candidates.length,1)
 const c=result.candidates[0], d=c.learning_data
 assert.equal(c.source_type,'employee_fact')
 assert.equal(d.proposal_version,2); assert.equal(d.review_required,true)
 assert.equal(d.managed_by_review,true); assert.equal(d.verification_status,'unverified')
 assert.equal(d.factual_use_approved,false);assert.equal(d.publish_state,'unpublished')
 assert.ok(d.limitations && d.evidence_quote && d.evidence_source && d.review_reason && c.retrieval_text)
})
test('incomplete, invented, identifying and operational facts are skipped without a review card',()=>{
 for(const c of [fact({limitations:''}),fact({evidence_quote:'Erfundene Büroregel'}),fact({fact_statement:'Wir öffnen um 8 Uhr.'}),fact({retrieval_text:'Mail person@example.org'}),fact({evidence_source:'raw_text'}),fact({fact_statement:'Ja',evidence_quote:'Ja, das ist korrekt.'})]) {
  const result=validate(comparison(),[c]);assert.equal(result.candidates.length,0);assert.ok(result.skip_reason)
 }
 const source=comparison({question:'Können Sie meine Bestellung stornieren?',human_response:'Wir stornieren Ihre Bestellung.'})
 assert.equal(validate(source,[fact({fact_statement:source.human_response,evidence_quote:source.human_response})]).candidates.length,0)
 assert.equal(validate(comparison(),[],'unclear').candidates.length,0)
})
test('safe style and explicit category corrections retain the automatic path',()=>{
 const style=validate(comparison({human_response:'Vielen Dank für Ihre Nachricht. Gerne sind wir für Sie da.'}),[{learning_type:'response_example',topic:'Freundlicher Ton',guidance:'Kurz und freundlich antworten',rationale:'Mitarbeiter bittet um eine kurze freundliche Antwort.'}],'style')
 assert.equal(style.candidates.length,1)
 assert.equal(style.candidates[0].learning_data.example_role,'style_only')
 assert.notEqual(style.candidates[0].learning_data.managed_by_review,true)
 const category={learning_type:'classification_example',topic:'Firmenanfrage',guidance:'Firmenfragen korrekt zuordnen',rationale:'Explizite Korrektur im Feedbackgrund',corrected_categories:['firmenfragen']}
 const result=validate(comparison({original_categories:['bestellung'],feedback_reason:'Keine Bestellung, sondern Firmenfrage'}),[category],'classification')
 assert.equal(result.candidates.length,1)
 assert.equal(result.candidates[0].learning_data.category_correction_origin,'explicit_reason')
 const guessed=validate(comparison({original_categories:['bestellung'],feedback_reason:'Mensch muss die Bestellung bearbeiten'}),[category],'classification')
 assert.equal(guessed.candidates.length,0)
 const handling=validate(comparison({original_categories:['bestellung'],feedback_reason:'Du darfst die Stornierung nicht ausführen, Mensch erforderlich'}),[{...category,corrected_categories:['human_needed']}],'classification')
 assert.equal(handling.candidates.length,0)
 const vague=validate(comparison({original_categories:['bestellung'],feedback_reason:'Die Antwort auf diese Firmenfrage ist falsch'}),[category],'classification')
 assert.equal(vague.candidates.length,0)
})
test('draft persistence locks feedback and deduplicates per feedback and learning type',()=>{
 const sql=v10node('Save Draft RAG Items').parameters.query
 assert.match(sql,/FOR UPDATE/);assert.match(sql,/f.learning_status = 'processing'/)
 assert.match(sql,/r.feedback_id = tf.id AND r.source_type = c.source_type/)
 assert.match(sql,/THEN 'prepared' ELSE 'skipped'/)
 for(const name of ['Claim RAG Publication','Finalize RAG Publication'])assert.match(v10node(name).parameters.query,/NOT EXISTS \(SELECT 1 FROM learning_reviews/)
})

test('prepared facts also carry the exact conservative Go operational guard',()=>{
 const model=readFileSync(new URL('../models.go',import.meta.url),'utf8')
 const pattern=model.match(/var operational = regexp.MustCompile\(`\(\?i\)(.*?)`\)/)[1]
 assert.ok(v10node('Validate Learning Candidates').parameters.jsCode.includes(JSON.stringify(pattern)))
})
